
#!/bin/bash
for filename in *.sql; do
    psql -u ${DATABASE_USER} -h ${DATABASE_HOST} -p ${DATABASE_NAME} < $filename
done
