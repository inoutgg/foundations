CREATE TABLE users (
  id UUID NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  email VARCHAR(255) NOT NULL,
  is_email_verified BOOLEAN NOT NULL DEFAULT FALSE,
  password_hash VARCHAR(4095) NULL,
  first_name VARCHAR(255) NULL,
  last_name VARCHAR(255) NULL,
  PRIMARY KEY (id),
  UNIQUE (email)
);

CREATE TABLE user_email_verification_tokens (
  id UUID NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  is_used BOOLEAN NOT NULL DEFAULT FALSE,
  token VARCHAR(16) NOT NULL,
  email VARCHAR(255) NOT NULL,
  user_id UUID NOT NULL,
  PRIMARY KEY (user_id, id),
  UNIQUE (email, is_used),
  UNIQUE (token),
  FOREIGN KEY (user_id) REFERENCES users (id)
    ON DELETE CASCADE
);

CREATE TABLE sso_provider_users (
  id UUID NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  provider_name VARCHAR(255) NOT NULL,
  user_id UUID NOT NULL,
  provider_user_id VARCHAR(255) NOT NULL, -- external user id
  PRIMARY KEY (user_id, id),
  UNIQUE (provider_name, user_id),
  UNIQUE (provider_name, provider_user_id),
  FOREIGN KEY (user_id) REFERENCES users (id)
);

CREATE TABLE password_reset_tokens (
  id UUID NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  is_used BOOLEAN NOT NULL DEFAULT FALSE,
  token VARCHAR(16) NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  user_id UUID NOT NULL,
  PRIMARY KEY (user_id, id),
  UNIQUE (token),
  UNIQUE (user_id, is_used),
  FOREIGN KEY (user_id) REFERENCES users (id)
    ON DELETE CASCADE,
  CHECK (expires_at > CURRENT_TIMESTAMP),
  CHECK (expires_at > created_at)
);

CREATE TABLE user_sessions (
  id UUID NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  expires_at TIMESTAMP NOT NULL,
  token VARCHAR(128) NOT NULL,
  user_id UUID NOT NULL,
  PRIMARY KEY (user_id, id),
  UNIQUE (token),
  FOREIGN KEY (user_id) REFERENCES users (id)
    ON DELETE CASCADE,
  CHECK (expires_at > CURRENT_TIMESTAMP),
  CHECK (expires_at > created_at)
);
