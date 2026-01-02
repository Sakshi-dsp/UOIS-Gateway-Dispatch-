# Start UOIS Gateway with .env file loading and live logs
# Usage: .\scripts\start_uois_gateway.ps1
# Press Ctrl+C to stop the server

$scriptRoot = $PSScriptRoot
$repoRoot = Split-Path $scriptRoot -Parent

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  UOIS Gateway - Starting Server" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# Check if .env file exists
$envFile = Join-Path $repoRoot ".env"
if (-not (Test-Path $envFile)) {
    Write-Host "Warning: .env file not found at $envFile" -ForegroundColor Yellow
    Write-Host "Creating .env from .env.example..." -ForegroundColor Yellow
    
    $envExample = Join-Path $repoRoot ".env.example"
    if (Test-Path $envExample) {
        Copy-Item $envExample $envFile
        Write-Host ".env file created from .env.example" -ForegroundColor Green
        Write-Host "Please update .env with your configuration values" -ForegroundColor Yellow
        Write-Host ""
    } else {
        Write-Host "Error: .env.example file not found. Please create .env manually." -ForegroundColor Red
        exit 1
    }
}

# Load environment variables from .env file
Write-Host "Loading environment variables from .env file..." -ForegroundColor Cyan
$envContent = Get-Content $envFile

foreach ($line in $envContent) {
    # Skip comments and empty lines
    $line = $line.Trim()
    if ($line -eq "" -or $line.StartsWith("#")) {
        continue
    }
    
    # Parse KEY=VALUE format
    if ($line -match "^([^=]+)=(.*)$") {
        $key = $matches[1].Trim()
        $value = $matches[2].Trim()
        
        # Remove quotes if present
        if ($value.StartsWith('"') -and $value.EndsWith('"')) {
            $value = $value.Substring(1, $value.Length - 2)
        }
        if ($value.StartsWith("'") -and $value.EndsWith("'")) {
            $value = $value.Substring(1, $value.Length - 2)
        }
        
        # Set environment variable
        [Environment]::SetEnvironmentVariable($key, $value, "Process")
    }
}

Write-Host "Environment variables loaded successfully" -ForegroundColor Green
Write-Host ""

# Display key configuration
Write-Host "Configuration Summary:" -ForegroundColor Cyan
Write-Host "  Server: $env:SERVER_HOST`:$env:SERVER_PORT" -ForegroundColor White
Write-Host "  Redis: $env:REDIS_HOST`:$env:REDIS_PORT" -ForegroundColor White
Write-Host "  PostgreSQL-E: $env:POSTGRES_E_HOST`:$env:POSTGRES_E_PORT" -ForegroundColor White
Write-Host "  Order Service: $env:ORDER_SERVICE_GRPC_HOST`:$env:ORDER_SERVICE_GRPC_PORT" -ForegroundColor White
Write-Host "  Admin Service: $env:ADMIN_SERVICE_GRPC_HOST`:$env:ADMIN_SERVICE_GRPC_PORT" -ForegroundColor White
Write-Host "  Log Level: $env:LOG_LEVEL" -ForegroundColor White
Write-Host ""

# Change to repo root
Push-Location $repoRoot

try {
    Write-Host "Starting UOIS Gateway server..." -ForegroundColor Yellow
    Write-Host "  - Health endpoint: http://$env:SERVER_HOST`:$env:SERVER_PORT/health" -ForegroundColor Gray
    Write-Host "  - Metrics endpoint: http://$env:SERVER_HOST`:$env:SERVER_PORT/metrics" -ForegroundColor Gray
    Write-Host "  - ONDC endpoints: http://$env:SERVER_HOST`:$env:SERVER_PORT/ondc/*" -ForegroundColor Gray
    Write-Host ""
    Write-Host "Press Ctrl+C to stop the server" -ForegroundColor Yellow
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host ""
    
    # Run the server (this will show live logs)
    go run cmd/server/main.go
} catch {
    Write-Host ""
    Write-Host "Error starting server: $_" -ForegroundColor Red
    exit 1
} finally {
    Pop-Location
    Write-Host ""
    Write-Host "Server stopped." -ForegroundColor Yellow
}

