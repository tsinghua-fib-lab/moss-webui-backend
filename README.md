# moss-webui-backend

The backend of the MOSS web UI.

## Pre-requisites

- PostgreSQL 14 or later: store the DBRecorder output of MOSS
- MongoDB 4.2 or later: store the map data

Attention: you should upload the map data to MongoDB using `mosstool.util.format_converter.pb2coll` manually to make sure the backend can get the map data and transform them into geojson data for the frontend.

## Run the backend

The backend uses environment variables to configure the connection to the database and the server port.
- `MONGO_URI`: the URI of the MongoDB server, e.g., `mongodb://localhost:27017`
- `MONGO_DB`: the name of the MongoDB database to store map data, e.g., `moss`
- `PG_URI`: the URI of the PostgreSQL server, e.g., `postgresql://localhost:5432`
- `PORT` (optional): the port of the server, e.g., `8080`

We recommend using docker to run the backend. You can build the docker image using the following command:

```bash
docker run --rm -p 8080:8080 -e MONGO_URI=mongodb://localhost:27017 -e MONGO_DB=moss -e PG_URI=postgresql://localhost:5432 moss-webui-backend
```

## API Docs

The backend uses Swagger to document the API. You can access the API docs by visiting `http(s)://<backend_url>/swagger/index.html` after running the backend.
