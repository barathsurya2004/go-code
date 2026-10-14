-- Nutrition Goals table
CREATE TABLE IF NOT EXISTS nutrition_goals (
    user_uuid VARCHAR(64) PRIMARY KEY,
    calories_target DOUBLE PRECISION NOT NULL DEFAULT 2200,
    protein_grams DOUBLE PRECISION NOT NULL DEFAULT 160,
    carbs_grams DOUBLE PRECISION NOT NULL DEFAULT 220,
    fat_grams DOUBLE PRECISION NOT NULL DEFAULT 65,
    fiber_grams DOUBLE PRECISION NOT NULL DEFAULT 30,
    sodium_mg DOUBLE PRECISION NOT NULL DEFAULT 2300,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Meal Logs table
CREATE TABLE IF NOT EXISTS meal_logs (
    id VARCHAR(64) PRIMARY KEY,
    user_uuid VARCHAR(64) NOT NULL,
    meal_type VARCHAR(32) NOT NULL,
    logged_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notes TEXT DEFAULT '',
    total_calories DOUBLE PRECISION NOT NULL DEFAULT 0,
    total_protein DOUBLE PRECISION NOT NULL DEFAULT 0,
    total_carbs DOUBLE PRECISION NOT NULL DEFAULT 0,
    total_fat DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_meal_logs_user_date ON meal_logs(user_uuid, logged_at);

-- Meal Items table
CREATE TABLE IF NOT EXISTS meal_items (
    id VARCHAR(64) PRIMARY KEY,
    meal_log_id VARCHAR(64) NOT NULL REFERENCES meal_logs(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    estimated_weight_grams DOUBLE PRECISION NOT NULL DEFAULT 0,
    calories DOUBLE PRECISION NOT NULL DEFAULT 0,
    protein_grams DOUBLE PRECISION NOT NULL DEFAULT 0,
    carbs_grams DOUBLE PRECISION NOT NULL DEFAULT 0,
    fat_grams DOUBLE PRECISION NOT NULL DEFAULT 0,
    fiber_grams DOUBLE PRECISION NOT NULL DEFAULT 0,
    sodium_mg DOUBLE PRECISION NOT NULL DEFAULT 0,
    confidence_score DOUBLE PRECISION NOT NULL DEFAULT 0.95,
    ingredients TEXT[] DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_meal_items_log_id ON meal_items(meal_log_id);

-- Nutrition AI Queue Jobs table
CREATE TABLE IF NOT EXISTS nutrition_jobs (
    job_id VARCHAR(64) PRIMARY KEY,
    user_uuid VARCHAR(64) NOT NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'PENDING',
    error_message TEXT DEFAULT '',
    meal_log_id VARCHAR(64) DEFAULT '',
    detected_items_json JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_nutrition_jobs_user ON nutrition_jobs(user_uuid, job_id);
