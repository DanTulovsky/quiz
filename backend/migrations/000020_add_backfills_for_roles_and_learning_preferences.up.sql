-- Add default roles and backfill role assignments and learning preferences
-- Idempotent inserts for existing databases
INSERT INTO roles (name, description)
VALUES ('user', 'Normal site access'),
    ('admin', 'Administrative access to all features') ON CONFLICT (name) DO NOTHING;
-- Assign 'user' role to all existing users
INSERT INTO user_roles (user_id, role_id)
SELECT u.id,
    r.id
FROM users u
    CROSS JOIN roles r
WHERE r.name = 'user'
    AND u.id NOT IN (
        SELECT user_id
        FROM user_roles
        WHERE role_id = r.id
    ) ON CONFLICT (user_id, role_id) DO NOTHING;
-- Assign 'admin' role to the admin user
INSERT INTO user_roles (user_id, role_id)
SELECT u.id,
    r.id
FROM users u
    CROSS JOIN roles r
WHERE u.username = 'admin'
    AND r.name = 'admin'
    AND u.id NOT IN (
        SELECT user_id
        FROM user_roles
        WHERE role_id = r.id
    ) ON CONFLICT (user_id, role_id) DO NOTHING;
-- Backfill user_learning_preferences for users missing a row
INSERT INTO user_learning_preferences (
        user_id,
        focus_on_weak_areas,
        include_review_questions,
        fresh_question_ratio,
        known_question_penalty,
        review_interval_days,
        weak_area_boost
    )
SELECT id,
    TRUE,
    TRUE,
    0.3,
    0.1,
    7,
    2.0
FROM users
WHERE id NOT IN (
        SELECT user_id
        FROM user_learning_preferences
    ) ON CONFLICT (user_id) DO NOTHING;
