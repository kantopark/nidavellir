FROM python:3.7.6-slim-stretch

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y curl apt-utils apt-transport-https debconf-utils gcc build-essential g++ unixodbc-dev

RUN curl https://packages.microsoft.com/keys/microsoft.asc | apt-key add -
RUN curl https://packages.microsoft.com/config/debian/10/prod.list > /etc/apt/sources.list.d/mssql-release.list
RUN apt-get update && ACCEPT_EULA=Y apt-get install -y msodbcsql17 mssql-tools

RUN pip install copulae numpy pandas pyodbc psycopg2-binary requests scikit-learn scipy sqlalchemy xlrd

ENV PATH="${PATH}:/opt/mssql-tools/bin"

WORKDIR /repo

# TEMPLATE LINE OVERWRITE

ENTRYPOINT ["python"]
