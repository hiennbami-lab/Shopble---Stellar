CREATE TABLE IF NOT EXISTS video (
  id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  video_url TEXT NOT NULL,
  video_idx VARCHAR(32) NOT NULL,
  platform VARCHAR(10) NOT NULL,
  original_srt_url VARCHAR(100) NOT NULL DEFAULT '',
  original_audio_url VARCHAR(100) NOT NULL DEFAULT '',
  subtitle_hub JSONB NOT NULL DEFAULT '{}',
  status SMALLINT NOT NULL DEFAULT 0,
  sub_status VARCHAR(10) NOT NULL DEFAULT '',
  create_time BIGINT NOT NULL,
  update_time BIGINT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uidx_video_idx(video_idx);
CREATE INDEX IF NOT EXISTS idx_video_status(video_idx, status);