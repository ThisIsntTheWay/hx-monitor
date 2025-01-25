FROM node:20-alpine AS build

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm install

COPY . ./
RUN npm run build

# -------------------------
FROM nginx:1.27.3-alpine

COPY --from=build /app/build /usr/share/nginx/html

EXPOSE 80

COPY preflight.sh /docker-entrypoint.d/90-preflight.sh