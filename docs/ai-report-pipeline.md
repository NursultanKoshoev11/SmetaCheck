# AI report pipeline

Rule: parser and checker are source of truth.

Flow:
1. User uploads estimate file.
2. Parser extracts normalized rows.
3. Checker creates deterministic issues.
4. Report module builds chart data from issues.
5. AI receives only parsed rows and issues.
6. AI returns summary and recommendations.

AI must not invent prices, rows or hidden calculations.

Supported future providers:
- OpenAI
- Gemini
- Anthropic
- Local LLM

Server will choose provider by env config.
