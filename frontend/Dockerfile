FROM node:20-alpine AS build

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm install
COPY . ./
RUN npm run build

RUN ls -alh

# -------------------------
FROM nginx:1.27.3-alpine
COPY --from=build /app/build /usr/share/nginx/html
RUN ls -alh /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]