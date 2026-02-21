<?php

/**
 * Legacy PHP application entry point.
 * In a real project, this would be a Symfony or Laravel app.
 * This simplified version demonstrates the migration pattern.
 */

declare(strict_types=1);

// Autoload (in real app: require __DIR__.'/../vendor/autoload.php')

header('Content-Type: application/json');
header('X-Served-By: php-legacy');

$uri = $_SERVER['REQUEST_URI'];
$method = $_SERVER['REQUEST_METHOD'];

// Simple routing
$route = parse_url($uri, PHP_URL_PATH);

// Database connection (shared with Go)
$dsn = getenv('DATABASE_URL') ?: 'pgsql:host=postgres;dbname=app';
try {
    $pdo = new PDO($dsn, 'postgres', 'postgres', [
        PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
    ]);
} catch (PDOException $e) {
    http_response_code(500);
    echo json_encode(['error' => 'Database connection failed']);
    exit;
}

// Route handlers
switch (true) {
    // Health check (will be migrated to Go)
    case $route === '/api/v1/health':
        echo json_encode([
            'status' => 'healthy',
            'service' => 'php-legacy',
            'version' => 'v1',
        ]);
        break;

    // List users - v1 (legacy)
    case $route === '/api/v1/users' && $method === 'GET':
        $stmt = $pdo->query('SELECT id, name, email, created_at FROM users ORDER BY id');
        $users = $stmt->fetchAll(PDO::FETCH_ASSOC);

        echo json_encode([
            'data' => $users,
            'meta' => [
                'version' => 'v1',
                'engine' => 'php',
            ],
        ]);
        break;

    // Create user - v1 (legacy)
    case $route === '/api/v1/users' && $method === 'POST':
        $input = json_decode(file_get_contents('php://input'), true);

        if (!isset($input['name']) || !isset($input['email'])) {
            http_response_code(400);
            echo json_encode(['error' => 'Name and email required']);
            break;
        }

        $stmt = $pdo->prepare('INSERT INTO users (name, email, created_at) VALUES (?, ?, NOW()) RETURNING id');
        $stmt->execute([$input['name'], $input['email']]);
        $id = $pdo->lastInsertId();

        http_response_code(201);
        echo json_encode([
            'id' => (int) $id,
            'message' => 'User created by PHP service',
        ]);
        break;

    // Legacy routes (to be migrated)
    case str_starts_with($route, '/legacy/'):
        echo json_encode([
            'message' => 'Legacy endpoint - to be migrated to Go',
            'path' => $route,
        ]);
        break;

    // Admin routes (low priority for migration)
    case str_starts_with($route, '/admin/'):
        echo json_encode([
            'message' => 'Admin endpoint - PHP only',
            'path' => $route,
        ]);
        break;

    // 404
    default:
        http_response_code(404);
        echo json_encode(['error' => 'Not found', 'path' => $route]);
}
