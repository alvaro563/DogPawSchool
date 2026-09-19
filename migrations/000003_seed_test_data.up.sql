-- ============================================================================
-- DogPaw - Migration: Seed Initial Admin User
-- ============================================================================

BEGIN;

INSERT INTO users (name, email, password, role, is_active)
VALUES (
    'Administrador',
    'admin@admin.com',
    '$2a$10$7e/17X94Y1y1aG502Q843.G7g16jS4iK0hYj6RrSGtkuhiG/OUlT6',
    'ADMIN',
    true
)
ON CONFLICT (email) DO NOTHING;

COMMIT;
