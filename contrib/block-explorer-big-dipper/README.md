## Big Dipper Block Explorer

The following is a guide for setting up and starting [BigDipper](https://bigdipper.live/) Block Explorer. The required dependencies are added as submodules.

- NOTE: The following guide is using Docker. Alternatively you can follow the [official docs](https://docs.bigdipper.live)

### Prerequisites

- Docker
- jq

### Install dependencies

``` sh
git submodule update --init --recursive # clone submodules
```

### Setup and start database

BigDipper backend (BDJuno a.k.a callisto) uses PostgreSQL database to persist data.

``` sh
docker-compose up -d postgres-db
```

- The database can be accessed with the following connection string: `postgresql://callisto:password@localhost:5432/callisto`

### Setup and start backend (BDJuno a.k.a callisto)

``` sh
curl http://0.0.0.0:26657/genesis | jq '.result.genesis' > ./config/genesis.json # fetch genesis from network RPC

docker-compose up -d callisto-backend
```

### Setup and start Hasura GraphQL engine

``` sh
docker-compose up -d hasura-graphql
```

### Setup and start frontend (Big Dipper)

``` sh
docker-compose up -d bigdipper-frontend
```

- The block explorer UI can be accessed at: <http://localhost:3001>
