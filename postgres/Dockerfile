FROM postgres:19-bookworm@sha256:f4859caf96de494e38efa69ed703b08a5bbd9605e152e604f2daec8841c87c41

# Install cron extension
RUN apt-get update && apt-get install -y postgresql-19-cron \
    && rm -rf /var/lib/apt/lists/*

# Tell Postgres to preload pg_cron
CMD ["postgres", "-c", "shared_preload_libraries=pg_cron",  "-c", "cron.database_name=jobs"]