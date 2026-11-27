package skillcatalog

import (
	"context"
	"fmt"
	"testing"

	skillmodels "github.com/kubeflow/hub/catalog/internal/catalog/skillcatalog/models"
	"github.com/stretchr/testify/require"
)

type failOnceSkillRepo struct {
	*fakeSkillRepo
	failName   string
	failDelete bool
}

func (r *failOnceSkillRepo) Save(s skillmodels.Skill) (skillmodels.Skill, error) {
	if *s.GetAttributes().Name == r.failName {
		r.failName = ""
		return nil, fmt.Errorf("temporary save failure")
	}
	return r.fakeSkillRepo.Save(s)
}

func (r *failOnceSkillRepo) DeleteByID(id int32) error {
	if r.failDelete {
		r.failDelete = false
		return fmt.Errorf("temporary deletion failure")
	}
	return r.fakeSkillRepo.DeleteByID(id)
}

type retryResolver struct {
	calls  int
	commit string
	paths  []string
}

func (r *retryResolver) Resolve(context.Context, SkillRepository, string, *Credentials) ([]ResolvedSkill, error) {
	r.calls++
	var skills []ResolvedSkill
	for _, path := range r.paths {
		s := resolvedAt(path)
		s.ResolvedCommit = r.commit
		skills = append(skills, s)
	}
	return skills, nil
}

func (r *retryResolver) RemoteCommit(context.Context, SkillRepository, string, *Credentials) (string, error) {
	return r.commit, nil
}

func TestSyncSourceLocked_RetriesPartialSaveFailure(t *testing.T) {
	for _, restart := range []bool{false, true} {
		for _, failPath := range []string{"skills/a", "skills/b", "skills/c"} {
			t.Run(fmt.Sprintf("restart=%t/fail=%s", restart, failPath), func(t *testing.T) {
				repo := newFakeSkillRepo()
				missing := skillEntityName("src", "https://example.com/a.git", failPath, "v1")
				res := &retryResolver{commit: "abc", paths: []string{"skills/a", "skills/b", "skills/c"}}
				l := newReconcileLoader(repo, res)
				l.services.SkillRepository = &failOnceSkillRepo{fakeSkillRepo: repo, failName: missing}
				l.syncSourceLocked(context.Background(), "src", specWithOneRepo())
				require.Len(t, repo.names(), 2)
				if restart {
					l = newReconcileLoader(repo, res)
				}
				l.syncSourceLocked(context.Background(), "src", specWithOneRepo())
				require.Contains(t, repo.names(), missing, "transient failure must recover on the next sync")
				require.Equal(t, 2, res.calls)
				l.syncSourceLocked(context.Background(), "src", specWithOneRepo())
				require.Equal(t, 2, res.calls, "a completely saved ref becomes cacheable again")
			})
		}
	}
}

func TestSyncSourceLocked_RetriesPartialUpdate(t *testing.T) {
	for _, change := range []string{"config", "commit"} {
		for _, failPath := range []string{"skills/a", "skills/b"} {
			t.Run(change+"/"+failPath, func(t *testing.T) {
				repo := newFakeSkillRepo()
				res := &retryResolver{commit: "abc", paths: []string{"skills/a", "skills/b"}}
				l := newReconcileLoader(repo, res)
				spec := specWithOneRepo()
				l.syncSourceLocked(context.Background(), "src", spec)
				if change == "config" {
					spec.Repositories[0].Category = "new"
				} else {
					res.commit = "def"
				}
				l.services.SkillRepository = &failOnceSkillRepo{fakeSkillRepo: repo,
					failName: skillEntityName("src", "https://example.com/a.git", failPath, "v1")}
				l.syncSourceLocked(context.Background(), "src", spec)
				// Recreate the loader to ensure recovery does not rely on memory-only state.
				l = newReconcileLoader(repo, res)
				l.syncSourceLocked(context.Background(), "src", spec)
				require.Equal(t, 3, res.calls)
				for _, s := range repo.byName {
					require.Equal(t, res.commit, skillProperty(s, propResolvedCommit))
					require.Equal(t, spec.Repositories[0].Category, skillProperty(s, propCategory))
				}
				l.syncSourceLocked(context.Background(), "src", spec)
				require.Equal(t, 3, res.calls)
			})
		}
	}
}

func TestSyncSourceLocked_RetriesOrphanDeletion(t *testing.T) {
	repo := newFakeSkillRepo()
	res := &retryResolver{commit: "abc", paths: []string{"skills/a", "skills/b"}}
	l := newReconcileLoader(repo, res)
	spec := specWithOneRepo()
	l.syncSourceLocked(context.Background(), "src", spec)
	spec.Repositories[0].IncludedSkills = []string{"a"}
	res.paths = []string{"skills/a"}
	l.services.SkillRepository = &failOnceSkillRepo{fakeSkillRepo: repo, failDelete: true}
	l.syncSourceLocked(context.Background(), "src", spec)
	require.Len(t, repo.names(), 2)
	l = newReconcileLoader(repo, res)
	l.syncSourceLocked(context.Background(), "src", spec)
	require.Len(t, repo.names(), 1)
	require.Equal(t, 3, res.calls)
	l.syncSourceLocked(context.Background(), "src", spec)
	require.Equal(t, 3, res.calls)
}

func TestSyncSourceLocked_ReindexesLegacyRowsOnce(t *testing.T) {
	repo := newFakeSkillRepo()
	res := &retryResolver{commit: "abc", paths: []string{"skills/a", "skills/b"}}
	l := newReconcileLoader(repo, res)
	spec := specWithOneRepo()
	l.syncSourceLocked(context.Background(), "src", spec)
	for _, s := range repo.byName {
		props := *s.GetProperties()
		filtered := props[:0]
		for _, p := range props {
			if p.Name != propRefSkillCount {
				filtered = append(filtered, p)
			}
		}
		*s.GetProperties() = filtered
	}
	l = newReconcileLoader(repo, res)
	l.syncSourceLocked(context.Background(), "src", spec)
	require.Equal(t, 2, res.calls)
	l.syncSourceLocked(context.Background(), "src", spec)
	require.Equal(t, 2, res.calls)
}

func TestIndexExistingByRef_RequiresCompleteConsistentRows(t *testing.T) {
	repository := specWithOneRepo().Repositories[0]
	cases := map[string]func([]skillmodels.Skill){
		"complete": func([]skillmodels.Skill) {},
		"missing count": func(rows []skillmodels.Skill) {
			props := rows[0].GetProperties()
			for i := range *props {
				if (*props)[i].Name == propRefSkillCount {
					(*props)[i].IntValue = nil
				}
			}
		},
		"mismatched count": func(rows []skillmodels.Skill) {
			props := rows[0].GetProperties()
			for i := range *props {
				if (*props)[i].Name == propRefSkillCount {
					(*props)[i].IntValue = i32Ptr(3)
				}
			}
		},
		"custom count is not internal": func(rows []skillmodels.Skill) {
			props := rows[0].GetProperties()
			for i := range *props {
				if (*props)[i].Name == propRefSkillCount {
					(*props)[i].IsCustomProperty = true
				}
			}
		},
		"mismatched commit": func(rows []skillmodels.Skill) {
			props := rows[0].GetProperties()
			for i := range *props {
				if (*props)[i].Name == propResolvedCommit {
					(*props)[i].StringValue = new("old")
				}
			}
		},
		"mismatched config": func(rows []skillmodels.Skill) {
			props := rows[0].GetProperties()
			for i := range *props {
				if (*props)[i].Name == propConfigDigest {
					(*props)[i].StringValue = new("old")
				}
			}
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			rows := []skillmodels.Skill{
				buildSkillEntity(resolvedAt("skills/a"), repository, "src", 2),
				buildSkillEntity(resolvedAt("skills/b"), repository, "src", 2),
			}
			mutate(rows)
			for range 2 {
				known, _ := indexExistingByRef(rows)
				if name == "complete" {
					require.Equal(t, "abc", known(repository, "v1"))
				} else {
					require.Empty(t, known(repository, "v1"))
				}
				rows[0], rows[1] = rows[1], rows[0]
			}
		})
	}
}
