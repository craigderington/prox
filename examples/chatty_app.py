#!/usr/bin/env python3
"""Test app that generates realistic logs continuously for testing log tailing"""
import logging
import time
import sys
import random
from datetime import datetime

# Configure logging with proper formatting
logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    datefmt='%Y-%m-%d %H:%M:%S',
    stream=sys.stdout
)

logger = logging.getLogger('ChattyApp')

# Realistic log messages
INFO_MESSAGES = [
    "Processing user request #{}",
    "Database query completed in {:.2f}ms",
    "Cache hit for key: user_{}",
    "HTTP GET /api/users/{} - 200 OK",
    "Background job completed successfully",
    "Websocket connection established for user_{}",
    "File uploaded: document_{}.pdf",
    "Session created for user_{}",
]

WARNING_MESSAGES = [
    "High memory usage detected: {:.1f}%",
    "Slow query detected: {:.2f}ms",
    "Rate limit approaching for IP: 192.168.1.{}",
    "Cache miss for key: session_{}",
    "Deprecated API endpoint called: /v1/users",
    "Retry attempt {} for failed operation",
]

ERROR_MESSAGES = [
    "Failed to connect to database: Connection timeout",
    "Exception in request handler: KeyError('user_id')",
    "Authentication failed for user_{}",
    "File not found: /tmp/upload_{}.tmp",
    "Invalid JSON in request body",
    "Queue processing failed: Message ID {}",
]

DEBUG_MESSAGES = [
    "SQL: SELECT * FROM users WHERE id = {}",
    "Redis GET: session:user:{}",
    "Calling external API: https://api.example.com/data",
    "Cache stats: {} hits, {} misses",
    "Worker thread {} processing job",
]

logger.info("🚀 Chatty App Started - Generating realistic logs every second...")

counter = 0
while True:
    counter += 1

    # Rotate through different log levels
    level = counter % 10

    if level == 0:
        # ERROR - 10% of logs
        msg = random.choice(ERROR_MESSAGES).format(counter)
        logger.error(msg)
    elif level in [1, 2]:
        # WARNING - 20% of logs
        template = random.choice(WARNING_MESSAGES)
        if '{}' in template:
            msg = template.format(random.randint(counter, counter + 100))
        else:
            msg = template
        logger.warning(msg)
    elif level == 3:
        # DEBUG - 10% of logs
        template = random.choice(DEBUG_MESSAGES)
        # Count number of format placeholders
        num_placeholders = template.count('{}')
        if num_placeholders == 1:
            msg = template.format(counter)
        elif num_placeholders == 2:
            msg = template.format(counter, counter + 1)
        else:
            msg = template
        logger.debug(msg)
    else:
        # INFO - 60% of logs
        msg = random.choice(INFO_MESSAGES).format(counter)
        logger.info(msg)

    # Occasionally simulate stack traces on errors
    if level == 0 and counter % 50 == 0:
        try:
            raise ValueError(f"Simulated error for demonstration #{counter}")
        except ValueError as e:
            logger.exception(f"Caught exception during processing:")

    sys.stdout.flush()
    sys.stderr.flush()
    time.sleep(1)
