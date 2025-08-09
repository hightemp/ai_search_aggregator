# Frontend build
FROM node:20-alpine AS builder
WORKDIR /app
COPY frontend/package.json frontend/vite.config.ts frontend/tsconfig*.json frontend/tailwind.config.cjs frontend/postcss.config.cjs ./
RUN npm install
COPY frontend ./frontend
RUN npm run build --prefix frontend

FROM nginx:alpine
COPY --from=builder /app/frontend/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
