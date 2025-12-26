#!/usr/bin/env python3
"""Python-themed test app that generates random Python interpreter and library logs continuously"""

import logging
import time
import sys
import random
from datetime import datetime

# Configure logging with Python-like formatting
logging.basicConfig(
    level=logging.DEBUG,
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
    stream=sys.stdout,
)

logger = logging.getLogger("PythonLogger")

# Python-themed log messages
PYTHON_INFO_MESSAGES = [
    "Python {}.{}.{} (tags/v{}.{}.{}, Dec  1 2023, 00:00:00) [MSC v.1937 64 bit (AMD64)] on win32",
    "Loading module '{}' from '/usr/lib/python3.11/{}'",
    "Importing module '{}' (version {})",
    "Byte-compiling {} to {}",
    "Running script '{}'",
    "Executing bytecode for module '{}'",
    "Creating namespace for package '{}'",
    "Found {} modules in package '{}'",
    "Installing package '{}' from {}",
    "Collecting {}=={}",
    "Successfully installed {}-{}",
]

PYTHON_WARNING_MESSAGES = [
    "DeprecationWarning: '{}' is deprecated, use '{}' instead",
    "PendingDeprecationWarning: '{}' will be removed in version {}",
    "ResourceWarning: unclosed file <_io.TextIOWrapper name='{}' mode='r' encoding='UTF-8'>",
    "UserWarning: {} is not a valid path",
    "SyntaxWarning: 'is' with a literal. Did you mean '=='?",
    "FutureWarning: {} is deprecated and will be removed in Python {}",
    "ImportWarning: Not importing directory '{}': missing __init__.py",
    "UnicodeWarning: Some characters could not be decoded",
]

PYTHON_ERROR_MESSAGES = [
    "ModuleNotFoundError: No module named '{}'",
    "SyntaxError: invalid syntax at line {} in file '{}'",
    "IndentationError: unexpected indent at line {} in '{}'",
    "NameError: name '{}' is not defined",
    "TypeError: {} argument must be {}, not '{}'",
    "ValueError: invalid literal for int() with base 10: '{}'",
    "AttributeError: '{}' object has no attribute '{}'",
    "ImportError: cannot import name '{}' from '{}'",
    "KeyError: '{}'",
    "IndexError: list index out of range",
]

PYTHON_DEBUG_MESSAGES = [
    "GC: collecting generation {}...",
    "Memory usage: {} bytes allocated",
    "Thread {} started",
    "Signal {} received",
    "Frame executed in {:.3f}ms",
    "Cache hit for module '{}'",
    "Optimizing bytecode for '{}'",
    "Resolving import: {} -> {}",
    "Loading extension module '{}'",
    "Initializing interpreter state",
]

PYTHON_LIBRARIES = [
    "os",
    "sys",
    "json",
    "logging",
    "threading",
    "multiprocessing",
    "asyncio",
    "sqlite3",
    "random",
    "math",
    "time",
    "datetime",
    "collections",
    "itertools",
    "functools",
    "operator",
    "pathlib",
    "subprocess",
    "tempfile",
    "shutil",
    "glob",
    "fnmatch",
    "linecache",
    "pickle",
    "copyreg",
    "copy",
    "pprint",
    "reprlib",
    "enum",
    "numbers",
    "fractions",
    "decimal",
    "contextlib",
    "warnings",
    "contextvars",
    "concurrent",
    "queue",
    "sched",
    "_thread",
    "dummy_thread",
    "io",
    "codecs",
    "unicodedata",
    "stringprep",
    "re",
    "difflib",
    "textwrap",
    "string",
    "binary",
    "struct",
    "weakref",
    "gc",
    "inspect",
    "site",
    "warnings",
    "contextlib",
    "atexit",
    "traceback",
    "future",
    "keyword",
    "ast",
    "symtable",
    "symbol",
    "token",
    "tokenize",
    "tabnanny",
    "pyclbr",
    "py_compile",
    "compileall",
    "dis",
    "pickletools",
    "platform",
    "errno",
    "ctypes",
    "msvcrt",
    "winreg",
    "winsound",
    "posix",
    "pwd",
    "spwd",
    "grp",
    "crypt",
    "termios",
    "tty",
    "pty",
    "fcntl",
    "pipes",
    "resource",
    "nis",
    "syslog",
    "optparse",
    "argparse",
    "getopt",
    "readline",
    "rlcompleter",
    "sqlite3",
    "zlib",
    "gzip",
    "bz2",
    "lzma",
    "zipfile",
    "tarfile",
    "csv",
    "configparser",
    "netrc",
    "xdrlib",
    "plistlib",
    "hashlib",
    "hmac",
    "secrets",
    "ssl",
    "socket",
    "mmap",
    "contextvars",
    "concurrent",
    "multiprocessing",
    "threading",
    "asyncio",
    "queue",
    "sched",
    "_thread",
    "dummy_thread",
    "io",
]

PYTHON_VERSIONS = ["3.8", "3.9", "3.10", "3.11", "3.12"]

logger.info(
    "🐍 Python Logger App Started - Generating Python-themed logs continuously..."
)

counter = 0
while True:
    counter += 1

    # Rotate through different log levels with Python-appropriate frequencies
    level = counter % 20

    if level == 0:
        # ERROR - 5% of logs (less frequent than generic app)
        template = random.choice(PYTHON_ERROR_MESSAGES)
        num_placeholders = template.count("{}")
        if num_placeholders == 1:
            arg = (
                random.choice(PYTHON_LIBRARIES)
                if "module" in template.lower()
                else f"var_{counter}"
            )
            msg = template.format(arg)
        elif num_placeholders == 2:
            if "argument must be" in template:
                msg = template.format("positional", "str")
            elif "object has no attribute" in template:
                msg = template.format(
                    random.choice(PYTHON_LIBRARIES), f"method_{counter}"
                )
            elif "cannot import name" in template:
                msg = template.format(
                    f"function_{counter}", random.choice(PYTHON_LIBRARIES)
                )
            else:
                msg = template.format(f"arg_{counter}", f"file_{counter}.py")
        else:
            msg = template
        logger.error(msg)

    elif level in [1, 2]:
        # WARNING - 10% of logs
        template = random.choice(PYTHON_WARNING_MESSAGES)
        num_placeholders = template.count("{}")
        if num_placeholders == 1:
            msg = template.format(random.choice(PYTHON_LIBRARIES))
        elif num_placeholders == 2:
            if "will be removed" in template:
                msg = template.format(
                    random.choice(PYTHON_LIBRARIES), random.choice(PYTHON_VERSIONS)
                )
            elif "use" in template:
                msg = template.format(f"old_{counter}", f"new_{counter}")
            else:
                msg = template.format(f"file_{counter}.py")
        else:
            msg = template
        logger.warning(msg)

    elif level in [3, 4]:
        # DEBUG - 10% of logs
        template = random.choice(PYTHON_DEBUG_MESSAGES)
        num_placeholders = template.count("{}")
        if num_placeholders == 1:
            if "module" in template:
                msg = template.format(random.choice(PYTHON_LIBRARIES))
            elif "Thread" in template:
                msg = template.format(counter)
            else:
                msg = template.format(random.uniform(0.001, 1.0))
        elif num_placeholders == 2:
            if "import" in template:
                msg = template.format(
                    random.choice(PYTHON_LIBRARIES),
                    f"/path/to/{random.choice(PYTHON_LIBRARIES)}",
                )
            else:
                msg = template.format(counter, counter * 1024)
        else:
            msg = template
        logger.debug(msg)

    else:
        # INFO - 75% of logs (more frequent - Python loading messages)
        template = random.choice(PYTHON_INFO_MESSAGES)
        num_placeholders = template.count("{}")
        if num_placeholders == 1:
            msg = template.format(random.choice(PYTHON_LIBRARIES))
        elif num_placeholders == 2:
            if "Byte-compiling" in template:
                msg = template.format(
                    f"/path/to/{random.choice(PYTHON_LIBRARIES)}.py",
                    f"/path/to/{random.choice(PYTHON_LIBRARIES)}.pyc",
                )
            elif "Successfully installed" in template:
                msg = template.format(
                    random.choice(PYTHON_LIBRARIES), random.choice(PYTHON_VERSIONS)
                )
            elif "Collecting" in template:
                msg = template.format(
                    random.choice(PYTHON_LIBRARIES), random.choice(PYTHON_VERSIONS)
                )
            else:
                msg = template.format(random.choice(PYTHON_LIBRARIES), counter)
        elif num_placeholders == 3:
            # Python version message
            version_parts = random.choice(PYTHON_VERSIONS).split(".")
            if len(version_parts) >= 2:
                major, minor = version_parts[0], version_parts[1]
                msg = template.format(major, minor, random.randint(1, 20), major, minor)
            else:
                msg = template.format(3, 11, random.randint(1, 20), 3, 11)
        else:
            msg = template
        logger.info(msg)

    # Occasionally simulate Python stack traces on errors
    if level == 0 and counter % 30 == 0:
        try:
            # Simulate a realistic Python error
            raise ImportError(f"No module named '{random.choice(PYTHON_LIBRARIES)}'")
        except ImportError as e:
            logger.exception(f"Exception during import:")

    # Occasionally show Python startup messages
    if counter % 100 == 1:
        logger.info(
            "Python {} on {}".format(
                random.choice(PYTHON_VERSIONS),
                random.choice(["linux", "darwin", "win32"]),
            )
        )

    sys.stdout.flush()
    sys.stderr.flush()
    time.sleep(0.8)  # Slightly faster than 1 second for more Python-like activity
