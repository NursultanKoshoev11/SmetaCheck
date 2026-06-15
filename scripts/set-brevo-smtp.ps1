$ErrorActionPreference = 'Stop'
Set-Location (Split-Path -Parent $PSScriptRoot)

Write-Host 'Brevo SMTP setup for SmetaCheck' -ForegroundColor Cyan
Write-Host 'Use the SMTP login and SMTP key from Brevo, not your Brevo account password.' -ForegroundColor Yellow

$smtpUsername = (Read-Host 'Brevo SMTP login').Trim()
if ([string]::IsNullOrWhiteSpace($smtpUsername)) {
    throw 'SMTP login cannot be empty.'
}

$securePassword = Read-Host 'Brevo SMTP key' -AsSecureString
$passwordPtr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($securePassword)
try {
    $smtpPassword = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($passwordPtr).Trim()
} finally {
    [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($passwordPtr)
}
if ([string]::IsNullOrWhiteSpace($smtpPassword)) {
    throw 'SMTP key cannot be empty.'
}

$senderEmail = (Read-Host 'Verified sender email in Brevo').Trim()
try {
    $parsedSender = [System.Net.Mail.MailAddress]::new($senderEmail)
} catch {
    throw 'Sender email is invalid.'
}
if ($parsedSender.Address -ne $senderEmail) {
    throw 'Enter only the sender email address.'
}

$envPath = Join-Path (Get-Location) '.env'
if (-not (Test-Path $envPath)) {
    throw ".env not found: $envPath"
}

$values = [ordered]@{
    SMTP_HOST = 'smtp-relay.brevo.com'
    SMTP_PORT = '587'
    SMTP_TLS_MODE = 'starttls'
    SMTP_TIMEOUT = '15s'
    SMTP_USERNAME = $smtpUsername
    SMTP_PASSWORD = $smtpPassword
    SMTP_FROM = $senderEmail
    SMTP_FROM_NAME = 'SmetaCheck KG'
}

$lines = Get-Content $envPath
$seen = @{}
$output = foreach ($line in $lines) {
    if ($line -match '^([^#][^=]*)=(.*)$') {
        $key = $matches[1].Trim()
        if ($values.Contains($key)) {
            $seen[$key] = $true
            "$key=$($values[$key])"
            continue
        }
    }
    $line
}
foreach ($key in $values.Keys) {
    if (-not $seen.ContainsKey($key)) {
        $output += "$key=$($values[$key])"
    }
}
Set-Content -Path $envPath -Value $output -Encoding utf8

Write-Host 'Brevo settings saved. Recreating API...' -ForegroundColor Green
docker compose up -d --force-recreate api
if ($LASTEXITCODE -ne 0) {
    throw 'Docker restart failed.'
}
Start-Sleep -Seconds 8

$providers = Invoke-RestMethod -Uri 'http://127.0.0.1:3000/api/v1/auth/providers' -TimeoutSec 20
Write-Host ("Email registration enabled: " + $providers.providers.email_registration) -ForegroundColor Cyan
Write-Host ("Password reset enabled: " + $providers.providers.password_reset) -ForegroundColor Cyan

$testRecipient = (Read-Host 'Optional test recipient email (press Enter to skip)').Trim()
if ($testRecipient) {
    try {
        $null = [System.Net.Mail.MailAddress]::new($testRecipient)
        $message = [System.Net.Mail.MailMessage]::new()
        $message.From = [System.Net.Mail.MailAddress]::new($senderEmail, 'SmetaCheck KG')
        $message.To.Add($testRecipient)
        $message.Subject = 'SmetaCheck SMTP test'
        $message.Body = 'Brevo SMTP is configured correctly for SmetaCheck.'
        $message.IsBodyHtml = $false

        $client = [System.Net.Mail.SmtpClient]::new('smtp-relay.brevo.com', 587)
        $client.EnableSsl = $true
        $client.Credentials = [System.Net.NetworkCredential]::new($smtpUsername, $smtpPassword)
        $client.Timeout = 20000
        $client.Send($message)
        $client.Dispose()
        $message.Dispose()
        Write-Host 'Test email sent successfully.' -ForegroundColor Green
    } catch {
        Write-Host ('Test email failed: ' + $_.Exception.Message) -ForegroundColor Red
        Write-Host 'Check that the sender is verified and the SMTP key is correct.' -ForegroundColor Yellow
    }
}

$smtpPassword = $null
Write-Host 'Brevo SMTP setup finished.' -ForegroundColor Green
Read-Host 'Press Enter to close'
