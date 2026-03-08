-- name: FileCreate :exec
INSERT INTO files (
    id, tenant_id, bucket, object_key, size, mime, owner_type, owner_id, created_at
) VALUES (
    sqlc.arg(id)::uuid,
    sqlc.narg(tenant_id)::uuid,
    sqlc.arg(bucket),
    sqlc.arg(object_key),
    sqlc.arg(size),
    sqlc.arg(mime),
    sqlc.arg(owner_type),
    sqlc.arg(owner_id)::uuid,
    sqlc.arg(created_at)
);
