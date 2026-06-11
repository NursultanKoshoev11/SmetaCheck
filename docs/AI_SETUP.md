# SmetaCheck AI analysis setup

## How it works

1. SmetaCheck stores the uploaded file.
2. The server parses XLSX, XLSM or CSV into normalized estimate rows.
3. Deterministic checks calculate arithmetic findings and risk counts.
4. The server sends only normalized rows and findings to the configured AI provider.
5. The provider returns a strict JSON report.
6. SmetaCheck validates the response, adds verified chart data, stores the report in PostgreSQL and reuses it for the same estimate.
7. If the provider is unavailable, SmetaCheck returns a deterministic rules-based report.

The binary source file and server file path are not sent to the provider.

## Enable OpenAI

Set these variables in the server `.env` file:

```env
AI_PROVIDER=openai
OPENAI_API_KEY=<server-side secret>
OPENAI_MODEL=gpt-4.1-mini
OPENAI_BASE_URL=https://api.openai.com/v1
AI_TIMEOUT=35s
AI_MAX_INPUT_ROWS=200
AI_MAX_OUTPUT_TOKENS=1800
```

Never add the real API key to Git, frontend variables, browser code or screenshots.

Restart the API after changing the configuration:

```bash
docker compose up -d --build api
```

## Disable external AI

```env
AI_PROVIDER=rules
OPENAI_API_KEY=
```

The application will continue to produce a deterministic report without external API calls.

## API response

`GET /v1/ai/estimate-summary/{estimate_id}` returns:

- `executive_brief`
- `risk_level`
- `data_quality_score`
- `key_risks`
- `priority_actions`
- `cost_flags`
- `questions`
- `recommendation`
- `chart_data`
- `analysis_source`
- `provider`
- `model`
- `prompt_version`
- `generated_at`
- optional `warning` when fallback is used

## Privacy and safety

- The request uses `store: false`.
- Only the authenticated estimate owner can request an analysis.
- Model output is restricted by a JSON schema.
- Graph values are calculated by SmetaCheck from deterministic findings, not invented by the model.
- Cached reports are linked to the owner, estimate, model, prompt version and input hash.
