# Excel input contract

SmetaCheck accepts real estimate spreadsheets.

Initial production parser expects the first sheet.
Columns:
1. work name
2. unit
3. quantity
4. unit price
5. total amount

Unsupported files must be rejected, not silently guessed.
