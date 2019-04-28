# Use a different base image?  This one is pretty big
FROM golang:1.12.4-stretch as base
LABEL maintainer="Justin Michalicek <jmichalicek@gmail.com>"

RUN apt-get update && apt-get install -y wget ca-certificates sudo vim && apt-get autoremove && apt-get clean
RUN echo 'deb http://apt.postgresql.org/pub/repos/apt/ stretch-pgdg main' >> /etc/apt/sources.list.d/pgdg.list
RUN wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add -
RUN apt-get update && apt-get install -y postgresql-client-11
RUN wget -qO- https://github.com/golang-migrate/migrate/releases/download/v4.3.1/migrate.linux-amd64.tar.gz | tar -zxv -C /go/bin/ --transform='s/migrate.linux-amd64/migrate/'
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
RUN go get golang.org/x/tools/cmd/stringer
# RUN wget https://github.com/golang/dep/releases/download/v0.3.2/dep-linux-amd64 -O /go/bin/dep
# Make a dev user rather than running as root?
# RUN chmod a+x /go/bin/dep
RUN chmod a+x /go/bin/migrate

RUN useradd -ms /bin/bash developer && echo "developer ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers
USER developer
RUN mkdir -p /go/src/github.com/jmichalicek/worrywort-server-go
WORKDIR /go/src/github.com/jmichalicek/worrywort-server-go
EXPOSE 8080

FROM base as builder
LABEL maintainer="Justin Michalicek <jmichalicek@gmail.com>"
COPY --chown=developer . /go/src/github.com/jmichalicek/worrywort-server-go/
WORKDIR /go/src/github.com/jmichalicek/worrywort-server-go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o ./dist/worrywortd ./cmd/worrywortd
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o ./dist/wortuser ./cmd/wortuser

FROM debian:stretch-slim as production
LABEL maintainer="Justin Michalicek <jmichalicek@gmail.com>"
RUN apt-get update && apt-get install -y wget ca-certificates sudo vim gnupg && apt-get autoremove && apt-get clean
RUN echo 'deb http://apt.postgresql.org/pub/repos/apt/ stretch-pgdg main' >> /etc/apt/sources.list.d/pgdg.list
RUN wget --quiet -O - https://www.postgresql.org/media/keys/ACCC4CF8.asc | apt-key add -
RUN apt-get update && apt-get install -y postgresql-client-11
RUN wget -qO- https://github.com/golang-migrate/migrate/releases/download/v4.3.1/migrate.linux-amd64.tar.gz | tar -zxv -C /usr/local/bin/ --transform='s/migrate.linux-amd64/migrate/'
RUN chmod a+x /usr/local/bin/migrate
RUN useradd -ms /bin/bash developer && echo "developer ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers
USER developer
WORKDIR /home/developer
COPY --chown=developer --from=builder /go/src/github.com/jmichalicek/worrywort-server-go/dist/* ./
# not 100% necessary, but makes life easier for now to have these
RUN mkdir _migrations
RUN mkdir _dev_seeds
COPY --chown=developer --from=builder /go/src/github.com/jmichalicek/worrywort-server-go/Makefile ./
COPY --chown=developer --from=builder /go/src/github.com/jmichalicek/worrywort-server-go/_migrations/* ./_migrations/
COPY --chown=developer --from=builder /go/src/github.com/jmichalicek/worrywort-server-go/_dev_seeds/* ./_dev_seeds/
CMD ["./worrywortd"]
