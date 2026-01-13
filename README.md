# go-dutch
A modern, lightweight alternative to Splitwise for managing group expenses and settling balances.

## Features
- **Group Management:** Create groups and invite members via unique tokens.
- **Expense Tracking:** Log expenses with detailed descriptions, multi-payer support, and customizable shares.
- **Auto-Settlement:** Intelligent balance calculation using a greedy algorithm to match debtors and creditors.
- **Multi-Currency:** Support for various currencies with usage tracking.
- **GraphQL API:** Clean, type-safe API with field-level resolvers for performance.
- **Rate Limiting:** Built-in protection against brute force and spam.

## Tech Stack
- **Backend:** Go (Golang)
- **API:** GraphQL (gqlgen)
- **Database:** PostgreSQL
- **SQL Generation:** sqlc (Type-safe SQL)
- **Infrastructure:** Docker, Docker Compose
- **Deployment:** Oracle Cloud (OCI) + GitHub Actions

## Local Development

### Prerequisites
- Go 1.24+
- Docker & Docker Compose
- [golang-migrate](https://github.com/golang-migrate/migrate) (optional, for local migrations)

### Setup
1. **Clone the repo:**
   ```bash
   git clone https://github.com/raythx98/go-dutch.git
   cd go-dutch
   ```

2. **Configure Environment:**
   Create a `.envrc` or `.env` file with the following:
   ```bash
   export APP_GODUTCH_DBUSERNAME=postgres
   export APP_GODUTCH_DBPASSWORD=password
   export APP_GODUTCH_DBHOST=localhost
   export APP_GODUTCH_DBPORT=5432
   export APP_GODUTCH_DBDEFAULTNAME=godutch
   export APP_GODUTCH_JWTSECRET=your_secret_key
   export APP_GODUTCH_SERVERPORT=8080
   ```

3. **Run with Docker Compose:**
   ```bash
   docker-compose up -d
   ```

4. **Run Locally (Development):**
   ```bash
   make run_local
   ```
   The API will be available at `http://localhost:8080/query` and the Playground at `http://localhost:8080/`.

## Infrastructure Setup (Oracle Cloud - OCI)

### 1. GitHub Secrets
Configure the following in your repository settings:
- `DEPLOY_HOST`: Public IP of your instance.
- `DEPLOY_USER`: Remote user (e.g., `opc`, `ubuntu`, `ec2-user`).
- `SSH_PRIVATE_KEY`: Your private SSH key.
- `DB_PASSWORD`: Password for the production Postgres container.
- `JWT_SECRET`: Secret for signing JWT tokens.

### 2. OCI Configuration
- **Instance:** Oracle Linux 8/9 or Ubuntu (Recommended: **VM.Standard.A1.Flex** for ARM performance).
- **Security List:** Allow Ingress for SSH (22), HTTP (80), HTTPS (443), and API Traffic (8080).

## API Documentation
This project uses GraphQL. You can explore the schema and test queries via the GraphQL Playground at the root URL (`/`) when running the server.

### Example: Get Group Balances
```graphql
query GetGroupExpenses($groupId: ID!) {
  expenses(groupId: $groupId) {
    expenses {
      name
      amount
      currency { code }
    }
    owes {
      user { name }
      amount
      currency { code }
    }
  }
}
```

## License
MIT
