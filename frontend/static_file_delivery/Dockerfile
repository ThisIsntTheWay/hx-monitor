FROM nginx:alpine

RUN cat <<EOF > /etc/nginx/nginx.conf
events {}

http {
    server {
        listen 80;

        location / {
            add_header Access-Control-Allow-Origin "*";
            root /usr/share/nginx/html;
            index index.html;
        }
    }
}
EOF

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
