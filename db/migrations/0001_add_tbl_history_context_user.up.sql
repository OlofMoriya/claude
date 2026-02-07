CREATE TABLE history (
  id SERIAL PRIMARY KEY,
  context_id INT NOT NULL,
  prompt VARCHAR(10000) NOT NULL,
  response VARCHAR(10000),
  responseContent VARCHAR(10000),
  abbreviation VARCHAR(1000), 
  token_count INT,
  user_id INT,
  created timestamp
  with
    time zone not null default now (),
    UNIQUE (id)
);

--alter table public.history owner to "owlllm";

CREATE INDEX idx_history_user_id ON history (user_id);
CREATE INDEX idx_history_context_id ON history (context_id);


CREATE TABLE Context (
  id SERIAL PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  user_id INT,
  created timestamp
  with
    time zone not null default now (),
    UNIQUE (id)
);

--alter table public.context owner to "owlllm";

CREATE INDEX idx_context_id ON context (id);
CREATE INDEX idx_context_created ON context (created);
CREATE INDEX idx_context_user_id ON context (user_id);

CREATE TABLE Users (
  id SERIAL PRIMARY KEY,
  name VARCHAR(50) NOT NULL,
  slack_id VARCHAR(50),
  email VARCHAR(50),
  created timestamp
  with
    time zone not null default now (),
    UNIQUE (id, email, slack_id)
);


--alter table public.users owner to "owlllm";
