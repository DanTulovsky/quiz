FROM node:18-alpine

# Install Artillery globally
RUN npm install -g artillery

# Install Artillery plugins
RUN npm install -g artillery-plugin-fuzzer

# Create app directory
WORKDIR /app

# Copy Artillery test files
COPY . .

# Set environment variables
ENV NODE_ENV=production
ENV NODE_OPTIONS="--no-deprecation"

# Default command
CMD ["artillery", "--help"]
