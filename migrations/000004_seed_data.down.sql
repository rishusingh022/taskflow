DELETE FROM tasks;
DELETE FROM projects;
DELETE FROM users WHERE email IN ('test@example.com', 'priya@example.com', 'amit@example.com');
