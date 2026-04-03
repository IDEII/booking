# scripts/run-load-test.ps1
param(
    [string]$K6Script = "tests/load_test.js",
    [string]$ServiceUrl = "http://localhost:8080"
)

Write-Host "Running load tests..." -ForegroundColor Yellow

# Check if k6 is installed
$k6 = Get-Command k6 -ErrorAction SilentlyContinue
if (-not $k6) {
    Write-Host "k6 not installed. Install from https://k6.io/docs/getting-started/installation/" -ForegroundColor Red
    exit 1
}

# Check if service is running
try {
    $response = Invoke-WebRequest -Uri "$ServiceUrl/dummyLogin" -Method POST -Body '{"role":"user"}' -ContentType "application/json" -UseBasicParsing -ErrorAction Stop
    if ($response.StatusCode -ne 200) {
        Write-Host "Service returned status $($response.StatusCode). Make sure it's running properly." -ForegroundColor Red
        exit 1
    }
} catch {
    Write-Host "Cannot connect to service at $ServiceUrl. Run 'make up' first." -ForegroundColor Red
    exit 1
}

# Check if test script exists
if (-not (Test-Path $K6Script)) {
    Write-Host "Test script not found: $K6Script" -ForegroundColor Red
    exit 1
}

# Run k6
Write-Host "Starting k6 load test..." -ForegroundColor Yellow
k6 run $K6Script

Write-Host "Load tests complete" -ForegroundColor Green