# docker run -it --name db -p 5432:5432 database
# sudo docker exec -it db psql -U user -d database
FROM postgres:latest

# Set environment variables
ENV POSTGRES_DB=database
ENV POSTGRES_USER=user
ENV POSTGRES_PASSWORD=password

# Copy initialization SQL scripts
COPY init.sql /docker-entrypoint-initdb.d/
# COPY seed.sql /docker-entrypoint-initdb.d/

# Expose PostgreSQL port
EXPOSE 5432
