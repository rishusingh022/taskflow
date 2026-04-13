-- Seed Users
-- Password for all users: "password123"
-- bcrypt cost 12 (verified with golang.org/x/crypto/bcrypt)
-- DO UPDATE ensures password is always correct even if migration ran with old hash
INSERT INTO users (id, name, email, password) VALUES
    ('d290f1ee-6c54-4b01-90e6-d701748f0851', 'Raju Kumar', 'test@example.com',
     '$2a$12$P.pGDer/fi9k1CaKXWTS5O.XSY5AMN0QbFm4lW1r6zZ5n3xDJXUi6'),
    ('d290f1ee-6c54-4b01-90e6-d701748f0852', 'Priya Sharma', 'priya@example.com',
     '$2a$12$P.pGDer/fi9k1CaKXWTS5O.XSY5AMN0QbFm4lW1r6zZ5n3xDJXUi6'),
    ('d290f1ee-6c54-4b01-90e6-d701748f0853', 'Amit Patel', 'amit@example.com',
     '$2a$12$P.pGDer/fi9k1CaKXWTS5O.XSY5AMN0QbFm4lW1r6zZ5n3xDJXUi6')
ON CONFLICT (email) DO UPDATE SET password = EXCLUDED.password;

-- Seed Projects
INSERT INTO projects (id, name, description, owner_id) VALUES
    ('a1b2c3d4-e5f6-7890-abcd-ef1234567890', 'Website Redesign',
     'Complete overhaul of the company marketing website for Q2 launch',
     'd290f1ee-6c54-4b01-90e6-d701748f0851'),
    ('a1b2c3d4-e5f6-7890-abcd-ef1234567891', 'Mobile App MVP',
     'Build the first version of our customer-facing mobile application',
     'd290f1ee-6c54-4b01-90e6-d701748f0852'),
    ('a1b2c3d4-e5f6-7890-abcd-ef1234567892', 'Internal Dashboard',
     'Analytics dashboard for the operations team to track KPIs',
     'd290f1ee-6c54-4b01-90e6-d701748f0851')
ON CONFLICT (id) DO NOTHING;

-- Seed Tasks for "Website Redesign"
INSERT INTO tasks (id, title, description, status, priority, project_id, assignee_id, due_date, created_by) VALUES
    ('b1c2d3e4-f5a6-7890-bcde-f12345678901', 'Design homepage mockup',
     'Create high-fidelity Figma designs for the new homepage including hero section and feature cards',
     'done', 'high',
     'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
     'd290f1ee-6c54-4b01-90e6-d701748f0852',
     '2026-04-15',
     'd290f1ee-6c54-4b01-90e6-d701748f0851'),

    ('b1c2d3e4-f5a6-7890-bcde-f12345678902', 'Implement responsive navbar',
     'Build a mobile-first navigation component with hamburger menu and dropdown support',
     'in_progress', 'medium',
     'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
     'd290f1ee-6c54-4b01-90e6-d701748f0851',
     '2026-04-20',
     'd290f1ee-6c54-4b01-90e6-d701748f0851'),

    ('b1c2d3e4-f5a6-7890-bcde-f12345678903', 'Write API documentation',
     'Document all REST endpoints with request/response examples using OpenAPI spec',
     'todo', 'low',
     'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
     NULL,
     '2026-04-30',
     'd290f1ee-6c54-4b01-90e6-d701748f0851'),

    ('b1c2d3e4-f5a6-7890-bcde-f12345678904', 'Set up CI/CD pipeline',
     'Configure GitHub Actions for automated testing and deployment to staging',
     'todo', 'high',
     'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
     'd290f1ee-6c54-4b01-90e6-d701748f0853',
     '2026-04-18',
     'd290f1ee-6c54-4b01-90e6-d701748f0851'),

    ('b1c2d3e4-f5a6-7890-bcde-f12345678905', 'Integrate payment gateway',
     'Connect Stripe for subscription billing on the pricing page',
     'in_progress', 'high',
     'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
     'd290f1ee-6c54-4b01-90e6-d701748f0851',
     '2026-04-22',
     'd290f1ee-6c54-4b01-90e6-d701748f0851')
ON CONFLICT (id) DO NOTHING;

-- Seed Tasks for "Mobile App MVP"
INSERT INTO tasks (id, title, description, status, priority, project_id, assignee_id, due_date, created_by) VALUES
    ('b1c2d3e4-f5a6-7890-bcde-f12345678906', 'User authentication flow',
     'Implement login, registration, and forgot password screens with form validation',
     'done', 'high',
     'a1b2c3d4-e5f6-7890-abcd-ef1234567891',
     'd290f1ee-6c54-4b01-90e6-d701748f0852',
     '2026-04-10',
     'd290f1ee-6c54-4b01-90e6-d701748f0852'),

    ('b1c2d3e4-f5a6-7890-bcde-f12345678907', 'Push notification setup',
     'Integrate Firebase Cloud Messaging for real-time push notifications',
     'todo', 'medium',
     'a1b2c3d4-e5f6-7890-abcd-ef1234567891',
     'd290f1ee-6c54-4b01-90e6-d701748f0853',
     '2026-04-25',
     'd290f1ee-6c54-4b01-90e6-d701748f0852'),

    ('b1c2d3e4-f5a6-7890-bcde-f12345678908', 'Offline data sync',
     'Enable offline mode with local SQLite cache and background sync when connectivity resumes',
     'in_progress', 'high',
     'a1b2c3d4-e5f6-7890-abcd-ef1234567891',
     'd290f1ee-6c54-4b01-90e6-d701748f0852',
     '2026-04-28',
     'd290f1ee-6c54-4b01-90e6-d701748f0852')
ON CONFLICT (id) DO NOTHING;

-- Seed Tasks for "Internal Dashboard"
INSERT INTO tasks (id, title, description, status, priority, project_id, assignee_id, due_date, created_by) VALUES
    ('b1c2d3e4-f5a6-7890-bcde-f12345678909', 'Design dashboard wireframes',
     'Create low-fidelity wireframes for the main dashboard layout and chart placements',
     'done', 'medium',
     'a1b2c3d4-e5f6-7890-abcd-ef1234567892',
     'd290f1ee-6c54-4b01-90e6-d701748f0851',
     '2026-04-05',
     'd290f1ee-6c54-4b01-90e6-d701748f0851'),

    ('b1c2d3e4-f5a6-7890-bcde-f12345678910', 'Build chart components',
     'Implement reusable bar chart and line chart components using Recharts library',
     'in_progress', 'high',
     'a1b2c3d4-e5f6-7890-abcd-ef1234567892',
     'd290f1ee-6c54-4b01-90e6-d701748f0853',
     '2026-04-18',
     'd290f1ee-6c54-4b01-90e6-d701748f0851')
ON CONFLICT (id) DO NOTHING;
