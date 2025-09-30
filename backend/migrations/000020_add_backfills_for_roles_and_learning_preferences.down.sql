-- Remove backfilled role assignments and user learning preferences rows
-- Note: This will NOT remove the roles themselves if other data depends on them
DELETE FROM user_roles
WHERE role_id IN (
        SELECT id
        FROM roles
        WHERE name IN ('user', 'admin')
    )
    AND user_id IN (
        SELECT id
        FROM users
    );
DELETE FROM user_learning_preferences
WHERE user_id IN (
        SELECT id
        FROM users
    )
    AND (
        focus_on_weak_areas = TRUE
        AND include_review_questions = TRUE
        AND fresh_question_ratio = 0.3
        AND known_question_penalty = 0.1
        AND review_interval_days = 7
        AND weak_area_boost = 2.0
    );
