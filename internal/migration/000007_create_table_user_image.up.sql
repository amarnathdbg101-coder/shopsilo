CREATE TABLE user_images (
    user_id INT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    image_url TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT NOW()
);
