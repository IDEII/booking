


import http from 'k6/http';
import { check, sleep } from 'k6';
import { Trend, Rate, Counter } from 'k6/metrics';


const responseTime = new Trend('response_time', true);
const errorRate = new Rate('errors');
const bookingSuccessRate = new Rate('booking_success');
const totalRequests = new Counter('total_requests');
const slotListDuration = new Trend('slot_list_duration', true);
const bookingDuration = new Trend('booking_duration', true);


export const options = {
  setupTimeout: '5m', 
  teardownTimeout: '1m',
  
  scenarios: {
    
    constant_load: {
      executor: 'constant-arrival-rate',
      rate: 100,
      timeUnit: '1s',
      duration: '600s', 
      preAllocatedVUs: 100,
      maxVUs: 200,
    },
    
    ramp_up: {
      executor: 'ramping-arrival-rate',
      startRate: 10,
      timeUnit: '1s',
      stages: [
        { duration: '30s', target: 50 },
        { duration: '30s', target: 100 },
        { duration: '30s', target: 200 },
      ],
      preAllocatedVUs: 150,
      maxVUs: 250,
      startTime: '600s',
    },
  },
  thresholds: {
    'http_req_duration': ['p(95)<200', 'p(99)<500'],
    'slot_list_duration': ['p(95)<200', 'p(99)<300'],
    'booking_duration': ['p(95)<300', 'p(99)<500'],
    'errors': ['rate<0.02'], 
    'booking_success': ['rate>0.90'], 
  },
};


let adminToken = '';
let rooms = [];
let allSlots = [];
let userTokens = [];


export function setup() {
  console.log('=== LOAD TEST SETUP (Optimized) ===');
  
  const baseUrl = 'http://localhost:8080';
  
  
  const adminRes = http.post(`${baseUrl}/dummyLogin`, JSON.stringify({ role: 'admin' }), {
    headers: { 'Content-Type': 'application/json' },
  });
  
  if (adminRes.status !== 200) {
    console.error(`Failed to get admin token: ${adminRes.status}`);
    return { error: 'Setup failed' };
  }
  
  adminToken = adminRes.json('token');
  console.log('✓ Admin token obtained');
  
  
  console.log('Creating 50 rooms...');
  const roomsCount = 50;
  
  
  for (let batch = 0; batch < 5; batch++) {
    const batchStart = batch * 10;
    const batchEnd = Math.min(batchStart + 10, roomsCount);
    
    
    const batchRequests = [];
    for (let i = batchStart; i < batchEnd; i++) {
      const roomName = `Room_${i+1}`;
      batchRequests.push({
        url: `${baseUrl}/rooms/create`,
        body: JSON.stringify({ 
          name: roomName, 
          description: `Test room ${i+1}`, 
          capacity: 10 + (i % 20) 
        }),
        params: { 
          headers: { 
            'Content-Type': 'application/json', 
            'Authorization': `Bearer ${adminToken}` 
          } 
        },
      });
    }
    
    
    const responses = http.batch(batchRequests);
    
    for (let j = 0; j < responses.length; j++) {
      const res = responses[j];
      if (res.status === 201) {
        const room = res.json('room');
        rooms.push({ id: room.id, name: room.name });
      }
    }
    
    console.log(`  Created ${rooms.length}/${roomsCount} rooms`);
    
    
    sleep(0.5);
  }
  
  console.log(`✓ Created ${rooms.length} rooms`);
  
  
  console.log('Creating schedules for all rooms...');
  
  const scheduleBatches = [];
  for (let i = 0; i < rooms.length; i += 10) {
    const batchRooms = rooms.slice(i, i + 10);
    const batchRequests = batchRooms.map(room => ({
      url: `${baseUrl}/rooms/${room.id}/schedule/create`,
      body: JSON.stringify({ 
        daysOfWeek: [1, 2, 3, 4, 5], 
        startTime: '09:00', 
        endTime: '18:00' 
      }),
      params: { 
        headers: { 
          'Content-Type': 'application/json', 
          'Authorization': `Bearer ${adminToken}` 
        } 
      },
    }));
    
    const responses = http.batch(batchRequests);
    scheduleBatches.push(...responses);
    
    if ((i + 10) % 20 === 0) {
      console.log(`  Created schedules for ${Math.min(i + 10, rooms.length)} rooms`);
    }
    
    sleep(0.3);
  }
  
  console.log('✓ All schedules created');
  
  
  console.log('Waiting for slot generation (3 seconds)...');
  sleep(3);
  
  
  console.log('Collecting available slots (sampling 20 rooms)...');
  const userRes = http.post(`${baseUrl}/dummyLogin`, JSON.stringify({ role: 'user' }), {
    headers: { 'Content-Type': 'application/json' },
  });
  const tempUserToken = userRes.json('token');
  
  const sampledRooms = rooms.slice(0, 20); 
  const today = new Date();
  let totalSlots = 0;
  
  for (const room of sampledRooms) {
    
    for (let dayOffset = 1; dayOffset <= 3; dayOffset++) {
      const date = new Date();
      date.setDate(today.getDate() + dayOffset);
      const dayOfWeek = date.getDay();
      
      if (dayOfWeek === 0 || dayOfWeek === 6) continue;
      
      const dateStr = date.toISOString().split('T')[0];
      
      const slotsRes = http.get(
        `${baseUrl}/rooms/${room.id}/slots/list?date=${dateStr}`,
        { headers: { 'Authorization': `Bearer ${tempUserToken}` } }
      );
      
      if (slotsRes.status === 200 && slotsRes.json('slots')) {
        const slots = slotsRes.json('slots');
        if (slots.length > 0) {
          
          const limitedSlots = slots.slice(0, 10);
          allSlots.push(...limitedSlots.map(slot => ({
            id: slot.id,
            roomId: room.id,
          })));
          totalSlots += limitedSlots.length;
        }
      }
    }
  }
  
  console.log(`✓ Collected ${allSlots.length} slots for testing`);
  
  
  console.log('Creating 1000 test users...');
  const usersCount = 1000;
  const usersPerBatch = 50;
  const batches = usersCount / usersPerBatch;
  
  for (let batch = 0; batch < batches; batch++) {
    const batchUsers = [];
    
    
    for (let i = 0; i < usersPerBatch; i++) {
      const userIndex = batch * usersPerBatch + i;
      const email = `loadtest_${userIndex}@example.com`;
      const password = 'password123';
      
      
      const registerRes = http.post(
        `${baseUrl}/register`,
        JSON.stringify({ email, password, role: 'user' }),
        { headers: { 'Content-Type': 'application/json' } }
      );
      
      if (registerRes.status === 201) {
        batchUsers.push({ email, password });
      }
    }
    
    
    const loginRequests = batchUsers.map(user => ({
      url: `${baseUrl}/login`,
      body: JSON.stringify({ email: user.email, password: user.password }),
      params: { headers: { 'Content-Type': 'application/json' } },
    }));
    
    const loginResponses = http.batch(loginRequests);
    
    for (let i = 0; i < loginResponses.length; i++) {
      const res = loginResponses[i];
      if (res.status === 200) {
        const token = res.json('token');
        userTokens.push(token);
      }
    }
    
    if ((batch + 1) % 10 === 0) {
      console.log(`  Created ${userTokens.length}/${usersCount} users`);
    }
    
    
    sleep(0.5);
  }
  
  console.log(`✓ Created ${userTokens.length} test users`);
  
  
  console.log('Pre-creating initial bookings...');
  let createdBookings = 0;
  const targetInitialBookings = Math.min(5000, allSlots.length * 10); 
  
  for (let i = 0; i < userTokens.length && createdBookings < targetInitialBookings; i++) {
    const userToken = userTokens[i];
    const bookingsToCreate = Math.min(5, targetInitialBookings - createdBookings);
    
    for (let b = 0; b < bookingsToCreate && createdBookings < targetInitialBookings; b++) {
      if (allSlots.length === 0) break;
      
      const randomSlot = allSlots[Math.floor(Math.random() * allSlots.length)];
      
      const createRes = http.post(
        `${baseUrl}/bookings/create`,
        JSON.stringify({ slotId: randomSlot.id, createConferenceLink: false }),
        { headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${userToken}` } }
      );
      
      if (createRes.status === 201) {
        createdBookings++;
      }
      
      
      if (b % 10 === 0) {
        sleep(0.05);
      }
    }
    
    if ((i + 1) % 100 === 0) {
      console.log(`  Created ${createdBookings} initial bookings`);
    }
  }
  
  console.log(`✓ Pre-created ${createdBookings} initial bookings`);
  console.log('=== SETUP COMPLETE ===');
  
  return { 
    adminToken, 
    userTokens: userTokens,
    rooms: sampledRooms,
    slots: allSlots,
    totalSlots: allSlots.length,
    totalUsers: userTokens.length,
  };
}


export default function(data) {
  if (data.error || !data.userTokens || data.userTokens.length === 0) {
    return;
  }
  
  totalRequests.add(1);
  
  
  const userToken = data.userTokens[Math.floor(Math.random() * data.userTokens.length)];
  
  
  const endpoint = Math.random();
  
  if (endpoint < 0.5) {
    
    testListSlots(data, userToken);
  } else if (endpoint < 0.75) {
    
    testListRooms(userToken);
  } else {
    
    testCreateAndCancelBooking(data, userToken);
  }
  
  
  sleep(Math.random() * 0.2);
}


function testListRooms(userToken) {
  const startTime = new Date();
  
  const res = http.get('http://localhost:8080/rooms/list', {
    headers: { 'Authorization': `Bearer ${userToken}` },
  });
  
  const duration = new Date() - startTime;
  responseTime.add(duration);
  
  const success = res.status === 200;
  errorRate.add(!success);
  
  check(res, {
    'rooms/list status is 200': (r) => r.status === 200,
  });
}


function testListSlots(data, userToken) {
  const startTime = new Date();
  
  if (!data.rooms || data.rooms.length === 0) return;
  const room = data.rooms[Math.floor(Math.random() * data.rooms.length)];
  
  
  const date = new Date();
  date.setDate(date.getDate() + Math.floor(Math.random() * 3) + 1);
  const dateStr = date.toISOString().split('T')[0];
  
  const res = http.get(
    `http://localhost:8080/rooms/${room.id}/slots/list?date=${dateStr}`,
    { headers: { 'Authorization': `Bearer ${userToken}` } }
  );
  
  const duration = new Date() - startTime;
  responseTime.add(duration);
  slotListDuration.add(duration);
  
  const success = res.status === 200;
  errorRate.add(!success);
  
  check(res, {
    'slots/list status is 200': (r) => r.status === 200,
  });
}


function testCreateAndCancelBooking(data, userToken) {
  if (!data.slots || data.slots.length === 0) return;
  
  const slot = data.slots[Math.floor(Math.random() * data.slots.length)];
  if (!slot || !slot.id) return;
  
  const startTime = new Date();
  
  const createRes = http.post(
    'http://localhost:8080/bookings/create',
    JSON.stringify({ slotId: slot.id, createConferenceLink: false }),
    { headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${userToken}` } }
  );
  
  const duration = new Date() - startTime;
  bookingDuration.add(duration);
  responseTime.add(duration);
  
  const createSuccess = createRes.status === 201;
  errorRate.add(!createSuccess);
  bookingSuccessRate.add(createSuccess);
  
  if (createSuccess) {
    const bookingId = createRes.json('booking.id');
    
    
    sleep(0.1);
    
    const cancelRes = http.post(
      `http://localhost:8080/bookings/${bookingId}/cancel`,
      null,
      { headers: { 'Authorization': `Bearer ${userToken}` } }
    );
    
    const cancelSuccess = cancelRes.status === 200;
    errorRate.add(!cancelSuccess);
  }
}


export function teardown(data) {
  console.log('\n=== LOAD TEST TEARDOWN ===');
  console.log(`Test users: ${data.totalUsers || 0}`);
  console.log(`Test rooms: ${data.rooms ? data.rooms.length : 0}`);
  console.log(`Test slots: ${data.totalSlots || 0}`);
  console.log('=== LOAD TEST FINISHED ===');
}


export function handleSummary(data) {
  const successRate = data.metrics.checks ? 
    (data.metrics.checks.values.passes / (data.metrics.checks.values.passes + data.metrics.checks.values.fails) * 100).toFixed(2) : 'N/A';
  
  return {
    'load_test_results.json': JSON.stringify(data, null, 2),
    stdout: `

                    LOAD TEST RESULTS                         

 Total Requests: ${data.metrics.total_requests?.values?.count || 0}                              
 Checks Passed: ${successRate}%                          

 HTTP Request Duration (p95): ${data.metrics.http_req_duration?.values['p(95)']?.toFixed(2) || 'N/A'}ms     
 Slot List Duration (p95): ${data.metrics.slot_list_duration?.values['p(95)']?.toFixed(2) || 'N/A'}ms       
 Booking Duration (p95): ${data.metrics.booking_duration?.values['p(95)']?.toFixed(2) || 'N/A'}ms         

 Error Rate: ${(data.metrics.errors?.values?.rate * 100 || 0).toFixed(2)}%                                    
 Booking Success Rate: ${(data.metrics.booking_success?.values?.rate * 100 || 0).toFixed(2)}%                       

    `,
  };
}