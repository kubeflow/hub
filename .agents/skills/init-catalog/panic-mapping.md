# Panic Recipes: Entity ↔ Schema Mapping (Panics 1–3)

These three panics handle the mapping layer between domain entities and database schema structs.
Use the **Type Reference Table** in SKILL.md to select the correct types for each entity.

## Panic 1: `mapEntityToSchema`

Map domain entity to database schema struct. Copy ID, TypeID, Name, ExternalID, timestamps with nil-safety.

For **context** entities (Name is `string`, non-pointer):
```go
func mapXxxToContext(entity models.Xxx) schema.Context {
    attrs := entity.GetAttributes()
    ctx := schema.Context{}
    if typeID := entity.GetTypeID(); typeID != nil {
        ctx.TypeID = *typeID
    }
    if entity.GetID() != nil {
        ctx.ID = *entity.GetID()
    }
    if attrs != nil {
        if attrs.Name != nil {
            ctx.Name = *attrs.Name
        }
        ctx.ExternalID = attrs.ExternalID
        if attrs.CreateTimeSinceEpoch != nil {
            ctx.CreateTimeSinceEpoch = *attrs.CreateTimeSinceEpoch
        }
        if attrs.LastUpdateTimeSinceEpoch != nil {
            ctx.LastUpdateTimeSinceEpoch = *attrs.LastUpdateTimeSinceEpoch
        }
    }
    return ctx
}
```

For **artifact** entities (Name is `*string`, pointer; also has URI and State):
```go
func mapXxxToArtifact(entity models.Xxx) schema.Artifact {
    attrs := entity.GetAttributes()
    art := schema.Artifact{}
    if typeID := entity.GetTypeID(); typeID != nil {
        art.TypeID = *typeID
    }
    if entity.GetID() != nil {
        art.ID = *entity.GetID()
    }
    if attrs != nil {
        art.Name = attrs.Name
        art.ExternalID = attrs.ExternalID
        if attrs.CreateTimeSinceEpoch != nil {
            art.CreateTimeSinceEpoch = *attrs.CreateTimeSinceEpoch
        }
        if attrs.LastUpdateTimeSinceEpoch != nil {
            art.LastUpdateTimeSinceEpoch = *attrs.LastUpdateTimeSinceEpoch
        }
    }
    return art
}
```

For **execution** entities (Name is `*string`, pointer):
```go
func mapXxxToExecution(entity models.Xxx) schema.Execution {
    attrs := entity.GetAttributes()
    exec := schema.Execution{}
    if typeID := entity.GetTypeID(); typeID != nil {
        exec.TypeID = *typeID
    }
    if entity.GetID() != nil {
        exec.ID = *entity.GetID()
    }
    if attrs != nil {
        exec.Name = attrs.Name
        if attrs.CreateTimeSinceEpoch != nil {
            exec.CreateTimeSinceEpoch = *attrs.CreateTimeSinceEpoch
        }
        if attrs.LastUpdateTimeSinceEpoch != nil {
            exec.LastUpdateTimeSinceEpoch = *attrs.LastUpdateTimeSinceEpoch
        }
    }
    return exec
}
```

## Panic 2: `mapSchemaToEntity`

Reverse mapping — build domain entity from schema + properties. Split properties into Properties and CustomProperties using the ToEntity mapper from the reference table.

```go
func mapSchemaToXxx(schemaEntity SCHEMA_TYPE, props []PROPERTY_TYPE) models.Xxx {
    entity := &models.XxxImpl{
        ID:     &schemaEntity.ID,
        TypeID: &schemaEntity.TypeID,
        Attributes: &models.XxxAttributes{
            Name:                     NAME_FIELD,  // &schemaEntity.Name for context, schemaEntity.Name for artifact/execution
            ExternalID:               schemaEntity.ExternalID,
            CreateTimeSinceEpoch:     &schemaEntity.CreateTimeSinceEpoch,
            LastUpdateTimeSinceEpoch: &schemaEntity.LastUpdateTimeSinceEpoch,
        },
    }

    properties := []dbmodels.Properties{}
    customProperties := []dbmodels.Properties{}
    for _, prop := range props {
        mapped := TO_ENTITY_MAPPER(prop)  // e.g. service.MapContextPropertyToProperties(prop)
        if prop.IsCustomProperty {
            customProperties = append(customProperties, mapped)
        } else {
            properties = append(properties, mapped)
        }
    }
    entity.Properties = &properties
    entity.CustomProperties = &customProperties
    return entity
}
```

For context entities, Name field is: `&schemaEntity.Name` (take address of string).
For artifact/execution entities, Name field is: `schemaEntity.Name` (already a pointer).

## Panic 3: `mapEntityToProperties`

Iterate entity properties and custom properties, convert each using the ToProperties mapper from the reference table.

```go
func mapXxxToProperties(entity models.Xxx, entityID int32) []PROPERTY_TYPE {
    var properties []PROPERTY_TYPE
    if entity.GetProperties() != nil {
        for _, prop := range *entity.GetProperties() {
            properties = append(properties, TO_PROPERTIES_MAPPER(prop, entityID, false))
        }
    }
    if entity.GetCustomProperties() != nil {
        for _, prop := range *entity.GetCustomProperties() {
            properties = append(properties, TO_PROPERTIES_MAPPER(prop, entityID, true))
        }
    }
    return properties
}
```
