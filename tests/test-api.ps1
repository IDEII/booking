


param(
    [string]$BaseUrl = "http://localhost:8080",
    [switch]$Verbose = $false
)


$ErrorColor = "Red"
$SuccessColor = "Green"
$InfoColor = "Cyan"
$WarningColor = "Yellow"

function Write-Test {
    param(
        [string]$TestName,
        [string]$Status,
        [string]$Message = ""
    )
    
    $timestamp = Get-Date -Format "HH:mm:ss"
    if ($Status -eq "PASS") {
        Write-Host "[$timestamp] ✓ $TestName" -ForegroundColor $SuccessColor
        if ($Message -and $Verbose) {
            Write-Host "    $Message" -ForegroundColor $SuccessColor
        }
    }
    elseif ($Status -eq "FAIL") {
        Write-Host "[$timestamp] ✗ $TestName" -ForegroundColor $ErrorColor
        if ($Message) {
            Write-Host "    $Message" -ForegroundColor $ErrorColor
        }
    }
    elseif ($Status -eq "INFO") {
        Write-Host "[$timestamp] ℹ $TestName" -ForegroundColor $InfoColor
        if ($Message) {
            Write-Host "    $Message" -ForegroundColor $InfoColor
        }
    }
    elseif ($Status -eq "WARN") {
        Write-Host "[$timestamp] ⚠ $TestName" -ForegroundColor $WarningColor
        if ($Message) {
            Write-Host "    $Message" -ForegroundColor $WarningColor
        }
    }
}

function Invoke-API {
    param(
        [string]$Method,
        [string]$Endpoint,
        [hashtable]$Body = $null,
        [string]$Token = $null
    )
    
    $uri = "$BaseUrl$Endpoint"
    $headers = @{}
    
    if ($Token) {
        $headers["Authorization"] = "Bearer $Token"
    }
    
    $params = @{
        Uri = $uri
        Method = $Method
        Headers = $headers
        ContentType = "application/json"
        ErrorAction = "Stop"
        TimeoutSec = 10
    }
    
    if ($Body) {
        $params["Body"] = ($Body | ConvertTo-Json -Compress)
    }
    
    try {
        $response = Invoke-WebRequest @params
        $content = $null
        if ($response.Content) {
            $content = $response.Content | ConvertFrom-Json
        }
        return @{
            Success = $true
            StatusCode = $response.StatusCode
            Content = $content
        }
    }
    catch {
        $errorContent = ""
        if ($_.Exception.Response) {
            $reader = New-Object System.IO.StreamReader($_.Exception.Response.GetResponseStream())
            $reader.BaseStream.Position = 0
            $reader.DiscardBufferedData()
            $errorContent = $reader.ReadToEnd()
            $reader.Close()
        }
        
        return @{
            Success = $false
            StatusCode = if ($_.Exception.Response) { $_.Exception.Response.StatusCode.value__ } else { 0 }
            Content = $errorContent
            Error = $_.Exception.Message
        }
    }
}

function Wait-ForSlots {
    param(
        [string]$Token,
        [string]$RoomId,
        [string]$Date,
        [int]$MaxRetries = 20,
        [int]$DelayMs = 500
    )
    
    Write-Test "  Waiting for slots on $Date..." -Status "INFO"
    
    for ($i = 0; $i -lt $MaxRetries; $i++) {
        $result = Invoke-API -Method "GET" -Endpoint "/rooms/$RoomId/slots/list?date=$Date" -Token $Token
        
        if ($result.Success -and $result.StatusCode -eq 200) {
            $slotCount = if ($result.Content.slots) { $result.Content.slots.Count } else { 0 }
            if ($slotCount -gt 0) {
                Write-Test "  Found $slotCount slots after $($i+1) attempts" -Status "PASS"
                return $result.Content.slots
            }
        }
        
        if ($i -lt $MaxRetries - 1) {
            Start-Sleep -Milliseconds $DelayMs
        }
    }
    
    Write-Test "  No slots available after $MaxRetries attempts" -Status "WARN"
    return $null
}

function Test-Info {
    Write-Test "Testing /_info" -Status "INFO"
    
    $result = Invoke-API -Method "GET" -Endpoint "/_info"
    
    if ($result.Success -and $result.StatusCode -eq 200) {
        Write-Test "/_info" -Status "PASS" -Message "OK"
        return $true
    }
    else {
        Write-Test "/_info" -Status "FAIL" -Message "Status: $($result.StatusCode)"
        return $false
    }
}

function Test-DummyLogin {
    Write-Test "Testing /dummyLogin" -Status "INFO"
    
    
    $adminResult = Invoke-API -Method "POST" -Endpoint "/dummyLogin" -Body @{role = "admin"}
    if ($adminResult.Success -and $adminResult.StatusCode -eq 200 -and $adminResult.Content.token) {
        Write-Test "  Admin login" -Status "PASS"
        $adminToken = $adminResult.Content.token
    }
    else {
        Write-Test "  Admin login" -Status "FAIL"
        return $null, $null
    }
    
    
    $userResult = Invoke-API -Method "POST" -Endpoint "/dummyLogin" -Body @{role = "user"}
    if ($userResult.Success -and $userResult.StatusCode -eq 200 -and $userResult.Content.token) {
        Write-Test "  User login" -Status "PASS"
        $userToken = $userResult.Content.token
    }
    else {
        Write-Test "  User login" -Status "FAIL"
        return $adminToken, $null
    }
    
    
    $invalidResult = Invoke-API -Method "POST" -Endpoint "/dummyLogin" -Body @{role = "invalid"}
    if (-not $invalidResult.Success -and $invalidResult.StatusCode -eq 400) {
        Write-Test "  Invalid role (400)" -Status "PASS"
    }
    else {
        Write-Test "  Invalid role" -Status "FAIL" -Message "Expected 400, got $($invalidResult.StatusCode)"
    }
    
    return $adminToken, $userToken
}

function Test-RegisterAndLogin {
    param([string]$AdminToken)
    
    Write-Test "Testing /register and /login" -Status "INFO"
    
    $testEmail = "test_$(Get-Date -Format 'yyyyMMddHHmmss')@example.com"
    $testPassword = "password123"
    
    
    $registerResult = Invoke-API -Method "POST" -Endpoint "/register" -Body @{
        email = $testEmail
        password = $testPassword
        role = "user"
    }
    
    if ($registerResult.Success -and $registerResult.StatusCode -eq 201) {
        Write-Test "  Register new user" -Status "PASS" -Message "Email: $testEmail"
        $userId = $registerResult.Content.user.id
    }
    else {
        Write-Test "  Register new user" -Status "FAIL" -Message "Status: $($registerResult.StatusCode)"
        return
    }
    
    
    $loginResult = Invoke-API -Method "POST" -Endpoint "/login" -Body @{
        email = $testEmail
        password = $testPassword
    }
    
    if ($loginResult.Success -and $loginResult.StatusCode -eq 200 -and $loginResult.Content.token) {
        Write-Test "  Login with credentials" -Status "PASS"
        $userToken = $loginResult.Content.token
    }
    else {
        Write-Test "  Login with credentials" -Status "FAIL"
    }
    
    
    $badLoginResult = Invoke-API -Method "POST" -Endpoint "/login" -Body @{
        email = $testEmail
        password = "wrongpassword"
    }
    
    if (-not $badLoginResult.Success -and $badLoginResult.StatusCode -eq 401) {
        Write-Test "  Invalid password (401)" -Status "PASS"
    }
    else {
        Write-Test "  Invalid password" -Status "FAIL"
    }
    
    
    $duplicateResult = Invoke-API -Method "POST" -Endpoint "/register" -Body @{
        email = $testEmail
        password = $testPassword
        role = "user"
    }
    
    if (-not $duplicateResult.Success -and $duplicateResult.StatusCode -eq 400) {
        Write-Test "  Duplicate email (400)" -Status "PASS"
    }
    else {
        Write-Test "  Duplicate email" -Status "FAIL"
    }
}

function Test-Rooms {
    param([string]$AdminToken, [string]$UserToken)
    
    Write-Test "Testing Rooms endpoints" -Status "INFO"
    
    $roomName = "Test Room $(Get-Date -Format 'HHmmss')"
    
    
    $createResult = Invoke-API -Method "POST" -Endpoint "/rooms/create" -Body @{
        name = $roomName
        description = "Test description"
        capacity = 10
    } -Token $AdminToken
    
    if ($createResult.Success -and $createResult.StatusCode -eq 201) {
        Write-Test "  Admin creates room" -Status "PASS" -Message "ID: $($createResult.Content.room.id)"
        $roomId = $createResult.Content.room.id
    }
    else {
        Write-Test "  Admin creates room" -Status "FAIL" -Message "Status: $($createResult.StatusCode)"
        return $null
    }
    
    
    $userCreateResult = Invoke-API -Method "POST" -Endpoint "/rooms/create" -Body @{
        name = "Should Fail"
    } -Token $UserToken
    
    if (-not $userCreateResult.Success -and $userCreateResult.StatusCode -eq 403) {
        Write-Test "  User cannot create room (403)" -Status "PASS"
    }
    else {
        Write-Test "  User cannot create room" -Status "FAIL" -Message "Expected 403, got $($userCreateResult.StatusCode)"
    }
    
    
    $adminListResult = Invoke-API -Method "GET" -Endpoint "/rooms/list" -Token $AdminToken
    if ($adminListResult.Success -and $adminListResult.StatusCode -eq 200) {
        $roomCount = if ($adminListResult.Content.rooms) { $adminListResult.Content.rooms.Count } else { 0 }
        Write-Test "  Admin lists rooms" -Status "PASS" -Message "Found $roomCount rooms"
    }
    else {
        Write-Test "  Admin lists rooms" -Status "FAIL"
    }
    
    
    $userListResult = Invoke-API -Method "GET" -Endpoint "/rooms/list" -Token $UserToken
    if ($userListResult.Success -and $userListResult.StatusCode -eq 200) {
        $roomCount = if ($userListResult.Content.rooms) { $userListResult.Content.rooms.Count } else { 0 }
        Write-Test "  User lists rooms" -Status "PASS" -Message "Found $roomCount rooms"
    }
    else {
        Write-Test "  User lists rooms" -Status "FAIL"
    }
    
    return $roomId
}

function Test-Schedule {
    param([string]$AdminToken, [string]$RoomId)
    
    Write-Test "Testing Schedule endpoints" -Status "INFO"
    
    
    $scheduleResult = Invoke-API -Method "POST" -Endpoint "/rooms/$RoomId/schedule/create" -Body @{
        daysOfWeek = @(1, 2, 3, 4, 5)
        startTime = "09:00"
        endTime = "18:00"
    } -Token $AdminToken
    
    if ($scheduleResult.Success -and $scheduleResult.StatusCode -eq 201) {
        Write-Test "  Create schedule" -Status "PASS" -Message "Days: Mon-Fri, 09:00-18:00"
        $scheduleId = $scheduleResult.Content.schedule.id
    }
    else {
        Write-Test "  Create schedule" -Status "FAIL" -Message "Status: $($scheduleResult.StatusCode)"
        return $false
    }
    
    
    $duplicateResult = Invoke-API -Method "POST" -Endpoint "/rooms/$RoomId/schedule/create" -Body @{
        daysOfWeek = @(1, 2, 3)
        startTime = "10:00"
        endTime = "17:00"
    } -Token $AdminToken
    
    if (-not $duplicateResult.Success -and $duplicateResult.StatusCode -eq 409) {
        Write-Test "  Duplicate schedule (409)" -Status "PASS"
    }
    else {
        Write-Test "  Duplicate schedule" -Status "FAIL" -Message "Expected 409, got $($duplicateResult.StatusCode)"
    }
    
    return $true
}

function Test-Slots {
    param([string]$UserToken, [string]$RoomId)
    
    Write-Test "Testing Slots endpoints" -Status "INFO"
    
    
    $targetDate = (Get-Date).AddDays(1)
    
    
    while ($targetDate.DayOfWeek -eq 'Saturday' -or $targetDate.DayOfWeek -eq 'Sunday') {
        $targetDate = $targetDate.AddDays(1)
    }
    
    $dateStr = $targetDate.ToString("yyyy-MM-dd")
    Write-Test "  Looking for slots on $dateStr ($($targetDate.DayOfWeek))" -Status "INFO"
    
    
    $slots = Wait-ForSlots -Token $UserToken -RoomId $RoomId -Date $dateStr -MaxRetries 20 -DelayMs 500
    
    if ($slots -and $slots.Count -gt 0) {
        $firstSlot = $slots[0]
        Write-Test "  Found $($slots.Count) slots, using first slot: $($firstSlot.id)" -Status "PASS"
        return $firstSlot.id
    }
    
    
    Write-Test "  No slots on $dateStr, trying next days..." -Status "WARN"
    
    for ($i = 1; $i -le 7; $i++) {
        $nextDate = $targetDate.AddDays($i)
        
        while ($nextDate.DayOfWeek -eq 'Saturday' -or $nextDate.DayOfWeek -eq 'Sunday') {
            $nextDate = $nextDate.AddDays(1)
        }
        
        $dateStr = $nextDate.ToString("yyyy-MM-dd")
        Write-Test "  Trying $dateStr ($($nextDate.DayOfWeek))" -Status "INFO"
        
        $slots = Wait-ForSlots -Token $UserToken -RoomId $RoomId -Date $dateStr -MaxRetries 15 -DelayMs 500
        
        if ($slots -and $slots.Count -gt 0) {
            $firstSlot = $slots[0]
            Write-Test "  Found $($slots.Count) slots on $dateStr" -Status "PASS"
            return $firstSlot.id
        }
    }
    
    Write-Test "  No slots available in next 7 days" -Status "FAIL"
    return $null
}

function Test-Bookings {
    param([string]$AdminToken, [string]$UserToken, [string]$RoomId, [string]$SlotId)
    
    Write-Test "Testing Bookings endpoints" -Status "INFO"
    
    if (-not $SlotId) {
        Write-Test "  No slot available, skipping booking tests" -Status "WARN"
        return
    }
    
    
    $createResult = Invoke-API -Method "POST" -Endpoint "/bookings/create" -Body @{
        slotId = $SlotId
        createConferenceLink = $false
    } -Token $UserToken
    
    if ($createResult.Success -and $createResult.StatusCode -eq 201) {
        Write-Test "  Create booking" -Status "PASS" -Message "ID: $($createResult.Content.booking.id)"
        $bookingId = $createResult.Content.booking.id
    }
    else {
        Write-Test "  Create booking" -Status "FAIL" -Message "Status: $($createResult.StatusCode)"
        return
    }
    
    
    Start-Sleep -Milliseconds 200
    
    
    $myBookingsResult = Invoke-API -Method "GET" -Endpoint "/bookings/my" -Token $UserToken
    if ($myBookingsResult.Success -and $myBookingsResult.StatusCode -eq 200) {
        $bookingsCount = if ($myBookingsResult.Content.bookings) { $myBookingsResult.Content.bookings.Count } else { 0 }
        Write-Test "  List my bookings" -Status "PASS" -Message "Found $bookingsCount bookings"
    }
    else {
        Write-Test "  List my bookings" -Status "FAIL"
    }
    
    
    $cancelResult = Invoke-API -Method "POST" -Endpoint "/bookings/$bookingId/cancel" -Token $UserToken
    if ($cancelResult.Success -and $cancelResult.StatusCode -eq 200) {
        Write-Test "  Cancel booking" -Status "PASS" -Message "Status: $($cancelResult.Content.booking.status)"
    }
    else {
        Write-Test "  Cancel booking" -Status "FAIL"
    }
}

function Test-ErrorCases {
    param([string]$AdminToken, [string]$UserToken, [string]$RoomId)
    
    Write-Test "Testing Error Cases" -Status "INFO"
    
    
    $unauthResult = Invoke-API -Method "GET" -Endpoint "/rooms/list"
    if (-not $unauthResult.Success -and $unauthResult.StatusCode -eq 401) {
        Write-Test "  Unauthorized access (401)" -Status "PASS"
    }
    else {
        Write-Test "  Unauthorized access" -Status "FAIL" -Message "Expected 401, got $($unauthResult.StatusCode)"
    }
    
    
    $fakeBookingId = [guid]::NewGuid()
    $fakeCancelResult = Invoke-API -Method "POST" -Endpoint "/bookings/$fakeBookingId/cancel" -Token $UserToken
    if (-not $fakeCancelResult.Success -and $fakeCancelResult.StatusCode -eq 404) {
        Write-Test "  Cancel non-existent booking (404)" -Status "PASS"
    }
    else {
        Write-Test "  Cancel non-existent booking" -Status "FAIL" -Message "Expected 404, got $($fakeCancelResult.StatusCode)"
    }
}


Write-Host ""
Write-Host "Waiting for service to start..." -ForegroundColor Yellow
$maxAttempts = 20
$attempt = 0
$serviceReady = $false

while ($attempt -lt $maxAttempts -and -not $serviceReady) {
    try {
        $response = Invoke-WebRequest -Uri "$BaseUrl/_info" -UseBasicParsing -TimeoutSec 2
        if ($response.StatusCode -eq 200) {
            $serviceReady = $true
            Write-Host "Service is ready!" -ForegroundColor Green
        }
    } catch {
        
    }
    
    if (-not $serviceReady) {
        $attempt++
        Write-Host "  Attempt $attempt/$maxAttempts..." -ForegroundColor Gray
        Start-Sleep -Seconds 1
    }
}

if (-not $serviceReady) {
    Write-Host "`n[ERROR] Service failed to start within $maxAttempts seconds" -ForegroundColor Red
    Write-Host "Check if containers are running with: docker-compose ps" -ForegroundColor Red
    exit 1
}


Write-Host ""
Write-Host ("=" * 60) -ForegroundColor $InfoColor
Write-Host "    API Testing Script - Room Booking Service" -ForegroundColor $InfoColor
Write-Host "    Base URL: $BaseUrl" -ForegroundColor $InfoColor
Write-Host "    Started: $(Get-Date)" -ForegroundColor $InfoColor
Write-Host ("=" * 60) -ForegroundColor $InfoColor


$adminToken, $userToken = Test-DummyLogin

if (-not $adminToken -or -not $userToken) {
    Write-Host "`n[ERROR] Failed to get tokens" -ForegroundColor $ErrorColor
    exit 1
}

Test-RegisterAndLogin -AdminToken $adminToken

$roomId = Test-Rooms -AdminToken $adminToken -UserToken $userToken

if ($roomId) {
    $scheduleCreated = Test-Schedule -AdminToken $adminToken -RoomId $roomId
    
    if ($scheduleCreated) {
        
        Write-Test "Waiting for slot generation to start..." -Status "INFO"
        Start-Sleep -Seconds 2
        
        $slotId = Test-Slots -UserToken $userToken -RoomId $roomId
        Test-Bookings -AdminToken $adminToken -UserToken $userToken -RoomId $roomId -SlotId $slotId
    }
}

Test-ErrorCases -AdminToken $adminToken -UserToken $userToken -RoomId $roomId


Write-Host ""
Write-Host ("=" * 60) -ForegroundColor $InfoColor
Write-Host "    Test Execution Completed: $(Get-Date)" -ForegroundColor $InfoColor
Write-Host ("=" * 60) -ForegroundColor $InfoColor
Write-Host ""