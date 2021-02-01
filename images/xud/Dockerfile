FROM node:lts-alpine3.12 AS builder
RUN apk add --no-cache git rsync bash musl-dev go python3 make g++
RUN ln -s /usr/bin/python3 /usr/bin/python
WORKDIR /opendexd
ADD .src .
ARG GIT_REVISION
RUN echo "" > parseGitCommit.js
RUN echo "export default '-$GIT_REVISION';" > lib/Version.ts
RUN npm install
RUN npm run compile
RUN npm run compile:seedutil
RUN npm prune --production
RUN rm -rf seedutil/go
RUN strip seedutil/seedutil

FROM node:lts-alpine3.12
RUN apk add --no-cache bash tor
COPY --from=builder /opendexd /app
COPY entrypoint.sh opendexd-backup.sh /
WORKDIR /app
RUN ln -s /app/bin/opendexd /usr/local/bin/opendexd
RUN ln -s /app/bin/opendexd-cli /usr/local/bin/opendexd-cli
ENTRYPOINT ["/entrypoint.sh"]
VOLUME [ "/root/.opendexd", "/root/backup" ]
EXPOSE 8887 18887 28887
