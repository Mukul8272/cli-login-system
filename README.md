# Containerized CLI Login System with Optional 2FA

A simple command-line authentication system built with **Go**, **MySQL**, and **Docker Compose**.

## Features

* User registration and login
* Secure password storage using bcrypt
* Optional TOTP-based 2FA
* Google Authenticator compatible
* Account lockout after multiple failed login attempts
* Configurable session timeout
* MySQL database with persistent storage
* Interactive command-line interface
* Command history
* Automatic database initialization

---

## Project Structure

```text
.
├── cmd/
│   └── app/
│       └── main.go
├── internal/
│   ├── auth/
│   │   └── service.go
│   ├── cli/
│   │   └── cli.go
│   ├── config/
│   │   └── config.go
│   ├── db/
│   │   └── mysql.go
│   └── models/
│       └── user.go
├── migrations/
│   └── 001_init.sql
├── Dockerfile
├── docker-compose.yml
├── .env.example
├── go.mod
├── go.sum
└── README.md
```

---

# How to Start the Project

## Step 1: Install Docker Desktop

Install **Docker Desktop** on the machine and make sure it is running.

open docker desktop

## Step 2: Get the Source Code

Clone or download the project source code.


Then open the project folder:

```bash
cd cli-login-system
```

---

## Step 3: Add the `.env` File

The actual `.env` file is provided **send saperately in email**.

Copy the `.env` file into the root of the project.

The structure should look like:

```text
cli-login-system/
├── .env
├── Dockerfile
├── docker-compose.yml
├── README.md
├── go.mod
├── go.sum
└── ...
```

## Step 4: Start the Application

Open a terminal inside the project folder and run:

```bash
docker compose build app
```

```bash
docker compose run --rm --interactive app
```

This will:

1. Start MySQL.
2. Wait for MySQL to become ready.
3. Build the application if required.
4. Start the interactive CLI.
5. Connect the application to MySQL.

You should see:

```text
login>
```

The application is now ready.

---

# Using the Application

## Before Login

Available commands:

```text
register
login
help
exit
```

### Register

Run:

```text
register
```

Enter a username and password.

Example:

```text
login> register
Username: mukul
Password: ********
```

After successful registration, you can log in.

### Login

Run:

```text
login
```

Enter your username and password.

After successful login, the application displays information such as:

```text
----------------------------------------
Username: mukul
Registration date: ...
MFA status: disabled
Session expiration time: ...
Last login time: ...
----------------------------------------
```

---

# Commands After Login

After logging in, the available commands are:

```text
whoami
enable-2fa
disable-2fa
logout
help
exit
```

### `whoami`

Displays the current user's information.

```text
mukul@cli> whoami
```

### `logout`

Ends the current session.

```text
mukul@cli> logout
```

### `exit`

Closes the CLI application.

```text
mukul@cli> exit
```

---

# Setting Up 2FA with Google Authenticator

The application uses **TOTP (Time-Based One-Time Password)** for two-factor authentication.

You can use **Google Authenticator** on your phone.

## Step 1: Login

Start the application:

```bash
docker compose run --rm --interactive app
```

Then log in:

```text
login> login
```

Enter your username and password.

---

## Step 2: Enable 2FA

After successful login, run:

```text
mukul@cli> enable-2fa
```

The application will display an `otpauth://` URL similar to:

```text
otpauth://totp/CLI%20Login%20System:mukul?algorithm=SHA1&digits=6&issuer=CLI%20Login%20System&period=30&secret=YOUR_SECRET
```

Keep this information private.

---

## Step 3: Open Google Authenticator

Install and open **Google Authenticator** on your phone.

Tap:

```text
+
```

Then select:

```text
Enter setup key
```

---

## Step 4: Enter the Setup Information

Enter an account name, for example:

```text
CLI Login System
```

From the `otpauth://` URL, find:

```text
secret=YOUR_SECRET
```

Copy only the value after `secret=`.

For example, if the URL contains:

```text
secret=ABC123XYZ456
```

enter:

```text
ABC123XYZ456
```

For the key type, select:

```text
Time based
```

Then tap **Add**.

Google Authenticator will now show a **6-digit verification code**.

The code changes periodically.

---

## Step 5: Logout

Return to the CLI and run:

```text
mukul@cli> logout
```

---

## Step 6: Login Again

Run:

```text
login> login
```

Enter:

```text
Username: mukul
Password: ********
TOTP code: 123456
```

Enter the current 6-digit code shown in Google Authenticator.

If the code is correct, login will be successful.

---

## Step 7: Disable 2FA

If you want to disable 2FA, first log in and run:

```text
mukul@cli> disable-2fa
```

After disabling it, TOTP will no longer be required during login.

---

## Important 2FA Security Note

The TOTP secret is sensitive.

Do not:

* Share the secret publicly.
* Commit the secret to GitHub/GitLab.
* Share screenshots containing the secret.

If the secret is exposed during testing, disable 2FA and enable it again to generate a new secret.

---

# Account Lockout

The application protects accounts against repeated failed login attempts.

The default settings are:

```text
Maximum failed attempts: 5
Lockout duration: 15 minutes
```

These values can be configured in `.env`:

```env
MAX_LOGIN_ATTEMPTS=5
LOCKOUT_MINUTES=15
```

For example:

```env
MAX_LOGIN_ATTEMPTS=3
LOCKOUT_MINUTES=10
```

means the account will be locked after 3 failed login attempts for 10 minutes.

---

# Session Timeout

The default session timeout is:

```text
30 minutes
```

It can be configured using:

```env
SESSION_TIMEOUT_MINUTES=30
```

After the session expires, the user must log in again.

---

# MySQL Database

The application uses **MySQL 8.4**.

The database runs inside Docker.

MySQL data is stored in a persistent Docker volume:

```text
mysql_data
```

This means the database data remains available when the application/container is restarted.

The database is automatically initialized when the application starts.

---

# Database Tables

The database contains:

### `users`

Stores user information such as:

* Username
* Password hash
* Registration date
* MFA status
* MFA secret
* Failed login attempts
* Lockout information
* Last login time

### `sessions`

Stores active session information and session expiration times.

The database schema is available in:

```text
migrations/001_init.sql
```

---

# Stopping the Project

To stop the Docker Compose services:

```bash
docker compose down
```

The MySQL data will remain in the `mysql_data` volume.

### Remove Database Data

To completely remove the containers **and the stored MySQL data**, run:

```bash
docker compose down -v
```

**Warning:** This permanently removes the Docker volume containing the database data.

---

# Rebuild the Application

If you make changes to the Go source code or Docker configuration, rebuild the application:

```bash
docker compose build --no-cache app
```

Then start it again:

```bash
docker compose run --rm --interactive app
```

---

# Troubleshooting

## Docker is not running

If you get an error similar to:

```text
The system cannot find the file specified
```

make sure Docker Desktop is running.

Test it with:

```bash
docker info
```

---

## CLI does not accept keyboard input

Make sure you start the application using:

```bash
docker compose run --rm --interactive app
```

Do not run the application in detached mode when using the interactive CLI.

---

## Check Docker containers

Run:

```bash
docker compose ps
```

---

## Check application logs

Run:

```bash
docker compose logs app
```

---

# Local Development

Docker is the recommended way to run the project.

For local development without Docker:

1. Install Go 1.23 or later.
2. Install and start MySQL.
3. Configure the required environment variables.
4. Download Go dependencies:

```bash
go mod tidy
```

5. Run the application:

```bash
go run ./cmd/app
```

---

# Security

The application includes the following security measures:

* Passwords are stored using bcrypt hashes.
* Passwords are never stored as plain text.
* Failed login attempts are tracked.
* Accounts can be temporarily locked.
* Sessions have configurable expiration.
* TOTP provides an additional authentication factor.
* Database configuration is provided through environment variables.
* The actual `.env` file should not be committed to source control.

---

# Quick Start

For a quick setup on a new machine:

### 1. Install and start Docker Desktop.

### 2. Clone/download the project.

### 3. Put the `.env` file provided separately by email into the project root.

### 4. Open a terminal in the project directory.

### 5. Run:

```bash
docker compose run --rm --interactive app
```

### 6. Register:

```text
register
```

### 7. Login:

```text
login
```

### 8. Enable 2FA:

```text
enable-2fa
```

### 9. Add the displayed secret to Google Authenticator.

### 10. Logout and login again.

### 11. Enter the 6-digit code from Google Authenticator when requested.

The application is now ready for testing.
