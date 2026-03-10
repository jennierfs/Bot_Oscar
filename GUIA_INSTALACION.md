# 📘 Guía de Instalación - Bot Oscar en VPS

Esta guía te ayudará a instalar y desplegar Bot Oscar en un servidor VPS (Virtual Private Server) paso a paso.

## 📋 Requisitos Previos

- Un VPS con Ubuntu 20.04 o superior (también funciona con Debian)
- Acceso SSH al servidor
- Al menos 2GB de RAM
- 20GB de espacio en disco
- Conexión a Internet

## 🚀 Paso 1: Conectarse al VPS

Abre tu terminal y conéctate a tu VPS:

```bash
ssh usuario@tu-ip-del-vps
```

Ejemplo:
```bash
ssh root@192.168.1.100
```

## 🔧 Paso 2: Actualizar el Sistema

Una vez dentro del VPS, actualiza los paquetes del sistema:

```bash
sudo apt update
sudo apt upgrade -y
```

## 🐳 Paso 3: Instalar Docker

Docker nos permite ejecutar la aplicación en contenedores aislados.

### 3.1 Instalar dependencias necesarias:

```bash
sudo apt install -y apt-transport-https ca-certificates curl software-properties-common
```

### 3.2 Agregar la clave GPG de Docker:

```bash
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
```

### 3.3 Agregar el repositorio de Docker:

```bash
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
```

### 3.4 Instalar Docker:

```bash
sudo apt update
sudo apt install -y docker-ce docker-ce-cli containerd.io
```

### 3.5 Verificar la instalación:

```bash
sudo docker --version
```

Deberías ver algo como: `Docker version 24.0.x`

### 3.6 Agregar tu usuario al grupo docker (opcional pero recomendado):

```bash
sudo usermod -aG docker $USER
newgrp docker
```

## 📦 Paso 4: Instalar Docker Compose

Docker Compose nos permite gestionar múltiples contenedores fácilmente.

```bash
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

Verificar la instalación:

```bash
docker-compose --version
```

## 📁 Paso 5: Subir el Proyecto al VPS

Tienes varias opciones:

### Opción A: Usando Git (Recomendado)

Si tu proyecto está en GitHub/GitLab:

```bash
cd /home
git clone https://github.com/tu-usuario/Bot_Oscar.git
cd Bot_Oscar
```

### Opción B: Usando SCP desde tu computadora local

Desde tu computadora (no en el VPS):

```bash
scp -r d:\Bot_Oscar usuario@tu-ip-del-vps:/home/
```

### Opción C: Usando SFTP

Puedes usar FileZilla o WinSCP para subir los archivos gráficamente.

## ⚙️ Paso 6: Configurar Variables de Entorno

Crea un archivo `.env` en la raíz del proyecto:

```bash
cd /home/Bot_Oscar
nano .env
```

Agrega las siguientes variables (ajusta según tus necesidades):

```env
# Base de Datos PostgreSQL
POSTGRES_USER=oscar
POSTGRES_PASSWORD=TuPasswordSeguro123!
POSTGRES_DB=trading_bot
POSTGRES_HOST=postgres
POSTGRES_PORT=5432

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=TuPasswordRedis123!

# API Keys (obtén estas claves registrándote en los servicios)
ALPHA_VANTAGE_API_KEY=tu_clave_aqui
TWELVE_DATA_API_KEY=tu_clave_aqui
DEEPSEEK_API_KEY=tu_clave_aqui

# Configuración del Backend
BACKEND_PORT=8080
ENVIRONMENT=production

# Configuración del Frontend
VITE_API_URL=http://tu-ip-del-vps:8080
```

Guarda el archivo presionando `Ctrl + X`, luego `Y`, y `Enter`.

### 📝 Cómo obtener las API Keys:

1. **Alpha Vantage**: Regístrate en https://www.alphavantage.co/support/#api-key
2. **Twelve Data**: Regístrate en https://twelvedata.com/pricing (plan gratuito disponible)
3. **DeepSeek**: Regístrate en https://platform.deepseek.com/

## 🔨 Paso 7: Construir y Ejecutar la Aplicación

### 7.1 Construir las imágenes Docker:

```bash
docker-compose build
```

Este proceso puede tardar 5-10 minutos la primera vez.

### 7.2 Iniciar los contenedores:

```bash
docker-compose up -d
```

El parámetro `-d` ejecuta los contenedores en segundo plano.

### 7.3 Verificar que los contenedores están corriendo:

```bash
docker-compose ps
```

Deberías ver algo como:

```
NAME                   STATUS              PORTS
bot_oscar_backend      Up 2 minutes        0.0.0.0:8080->8080/tcp
bot_oscar_frontend     Up 2 minutes        0.0.0.0:80->80/tcp
bot_oscar_postgres     Up 2 minutes        5432/tcp
bot_oscar_redis        Up 2 minutes        6379/tcp
```

## 🌐 Paso 8: Configurar el Firewall

Permite el tráfico en los puertos necesarios:

```bash
sudo ufw allow 80/tcp      # Frontend
sudo ufw allow 8080/tcp    # Backend API
sudo ufw allow 22/tcp      # SSH
sudo ufw enable
```

Confirma con `y` cuando se solicite.

## ✅ Paso 9: Verificar la Instalación

### 9.1 Verificar el Backend:

```bash
curl http://localhost:8080/health
```

O desde tu navegador: `http://tu-ip-del-vps:8080/health`

### 9.2 Verificar el Frontend:

Abre tu navegador y visita: `http://tu-ip-del-vps`

Deberías ver la interfaz de Bot Oscar.

### 9.3 Ver los logs:

```bash
# Ver logs de todos los servicios
docker-compose logs

# Ver logs del backend
docker-compose logs backend

# Ver logs en tiempo real
docker-compose logs -f backend
```

## 🔧 Comandos Útiles

### Detener la aplicación:
```bash
docker-compose down
```

### Reiniciar la aplicación:
```bash
docker-compose restart
```

### Actualizar la aplicación:
```bash
git pull                    # Si usas Git
docker-compose down
docker-compose build --no-cache
docker-compose up -d
```

### Ver el estado de los contenedores:
```bash
docker-compose ps
```

### Entrar a un contenedor:
```bash
docker-compose exec backend sh
docker-compose exec postgres psql -U oscar -d trading_bot
```

### Limpiar recursos de Docker:
```bash
docker system prune -a
```

## 🔐 Paso 10: Seguridad Adicional (Recomendado)

### 10.1 Configurar un dominio con SSL (HTTPS)

Si tienes un dominio, puedes configurar HTTPS con Let's Encrypt:

```bash
sudo apt install -y certbot python3-certbot-nginx
```

### 10.2 Cambiar el puerto SSH por defecto:

```bash
sudo nano /etc/ssh/sshd_config
```

Cambia `Port 22` por otro puerto (ej: `Port 2222`), guarda y reinicia:

```bash
sudo systemctl restart sshd
```

### 10.3 Deshabilitar login de root:

En el mismo archivo `/etc/ssh/sshd_config`, cambia:
```
PermitRootLogin no
```

## 🐛 Solución de Problemas Comunes

### Problema: Los contenedores no inician

```bash
# Ver logs detallados
docker-compose logs

# Reconstruir sin caché
docker-compose build --no-cache
docker-compose up -d
```

### Problema: No puedo conectarme desde el navegador

1. Verifica que el firewall permita el tráfico:
   ```bash
   sudo ufw status
   ```

2. Verifica que los contenedores están corriendo:
   ```bash
   docker-compose ps
   ```

3. Verifica la IP de tu VPS:
   ```bash
   curl ifconfig.me
   ```

### Problema: Error de conexión a la base de datos

```bash
# Verifica que PostgreSQL está corriendo
docker-compose exec postgres pg_isready

# Reinicia los servicios
docker-compose restart
```

### Problema: El frontend no se conecta al backend

1. Verifica que `VITE_API_URL` en el archivo `.env` tenga la IP correcta
2. Reconstruye el frontend:
   ```bash
   docker-compose build frontend
   docker-compose up -d
   ```

## 📊 Monitoreo

### Ver uso de recursos:

```bash
docker stats
```

### Ver logs en tiempo real:

```bash
docker-compose logs -f
```

## 🔄 Actualizaciones Automáticas (Opcional)

Para reiniciar automáticamente los contenedores si se caen:

Edita el archivo `docker-compose.yml` y agrega a cada servicio:

```yaml
restart: always
```

## 📞 Soporte

Si tienes problemas, revisa los logs:

```bash
docker-compose logs backend
docker-compose logs frontend
```

---

## 🎉 ¡Listo!

Tu Bot Oscar está ahora desplegado y corriendo en tu VPS. Puedes acceder a él desde cualquier navegador usando la IP de tu servidor.

**URL Frontend:** `http://tu-ip-del-vps`  
**URL API:** `http://tu-ip-del-vps:8080`

### Próximos Pasos Recomendados:

1. ✅ Configurar un dominio personalizado
2. ✅ Implementar HTTPS con Let's Encrypt
3. ✅ Configurar backups automáticos de la base de datos
4. ✅ Configurar monitoreo con Prometheus/Grafana
5. ✅ Implementar CI/CD con GitHub Actions

---

**Nota:** Recuerda mantener tus API keys seguras y nunca compartirlas públicamente.
