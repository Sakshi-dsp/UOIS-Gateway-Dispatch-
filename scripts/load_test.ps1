# Load Testing Script for UOIS Gateway (PowerShell)
# Tests throughput requirements: minimum 1000 requests/second

param(
    [string]$GatewayUrl = "http://localhost:8080",
    [string]$ClientId = "test-client",
    [string]$ClientSecret = "test-secret",
    [string]$Endpoint = "/ondc/status",
    [int]$Rate = 1000,
    [string]$Duration = "60s"
)

Write-Host "Starting load test..."
Write-Host "Gateway URL: $GatewayUrl"
Write-Host "Endpoint: $Endpoint"
Write-Host "Rate: $Rate req/s"
Write-Host "Duration: $Duration"
Write-Host ""

# Check if vegeta is installed
$vegetaPath = Get-Command vegeta -ErrorAction SilentlyContinue
if (-not $vegetaPath) {
    Write-Host "Error: vegeta is not installed"
    Write-Host "Install with: go install github.com/tsenart/vegeta/v12@latest"
    exit 1
}

# Create auth header
$credentials = "$ClientId`:$ClientSecret"
$bytes = [System.Text.Encoding]::UTF8.GetBytes($credentials)
$authHeader = [Convert]::ToBase64String($bytes)

# Create test payload
$timestamp = Get-Date -Format "yyyy-MM-ddTHH:mm:ss.000Z" -AsUTC
$txnId = "test-txn-$(Get-Date -UFormat %s)"
$msgId = "test-msg-$(Get-Date -UFormat %s)"
$payload = @{
    context = @{
        domain = "nic2004:60221"
        country = "IND"
        city = "std:080"
        action = "status"
        core_version = "1.2.0"
        bap_id = "test-bap"
        bap_uri = "https://test-bap.com"
        transaction_id = $txnId
        message_id = $msgId
        timestamp = $timestamp
        ttl = "PT30S"
    }
    message = @{
        order_id = "test-order-123"
    }
} | ConvertTo-Json -Depth 10

# Create vegeta target file
$targetFile = New-TemporaryFile
$targetContent = @"
POST $GatewayUrl$Endpoint
Authorization: Basic $authHeader
Content-Type: application/json

$payload
"@
Set-Content -Path $targetFile.FullName -Value $targetContent

try {
    Write-Host "Running load test..."
    & vegeta attack -rate=$Rate -duration=$Duration -targets="$($targetFile.FullName)" | `
        & vegeta report -type=text

    Write-Host ""
    Write-Host "Generating detailed report..."
    & vegeta attack -rate=$Rate -duration=$Duration -targets="$($targetFile.FullName)" | `
        & vegeta report -type=json | Out-File -FilePath "load_test_results.json" -Encoding utf8

    Write-Host "Results saved to load_test_results.json"
    Write-Host ""
    Write-Host "To view latency distribution:"
    Write-Host "  vegeta attack -rate=$Rate -duration=$Duration -targets=`"$($targetFile.FullName)`" | vegeta report -type=hist[0,100ms,200ms,500ms,1s,2s]"
} finally {
    Remove-Item $targetFile.FullName -ErrorAction SilentlyContinue
}

