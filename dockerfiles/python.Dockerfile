FROM python:3.7.5-slim-stretch

RUN apt-get update && \
    apt-get upgrade -y && \
    apt-get install -y curl apt-utils apt-transport-https debconf-utils gcc build-essential g++ unixodbc-dev

RUN curl https://packages.microsoft.com/keys/microsoft.asc | apt-key add -
RUN curl https://packages.microsoft.com/config/debian/10/prod.list > /etc/apt/sources.list.d/mssql-release.list
RUN apt-get update && ACCEPT_EULA=Y apt-get install -y msodbcsql17

RUN pip install numpy scipy pandas pyodbc scikit-learn copulae pyodbc psycopg2-binary requests xlrd

WORKDIR /repo

# TEMPLATE LINE OVERWRITE

ENTRYPOINT ["python"]
