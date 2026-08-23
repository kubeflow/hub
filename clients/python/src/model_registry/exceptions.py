"""Exceptions for the model registry."""


class StoreError(Exception):
    """Storage related error."""


class MissingMetadata(Exception):
    """Not enough metadata to complete operation."""


class UnsupportedType(StoreError):
    """Raised when an unsupported type is encountered."""


class TypeNotFound(StoreError):
    """Raised when a type cannot be found."""


class ServerError(StoreError):
    """Raised when the server returns a bad response."""


class DuplicateError(StoreError):
    """Raised when the user tries to put an object with a conflicting property."""
