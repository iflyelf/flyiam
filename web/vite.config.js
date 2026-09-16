import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  
  return {
    plugins: [vue()],
    
    resolve: {
      alias: {
        '@': resolve(__dirname, 'src')
      }
    },
    
    server: {
      port: parseInt(env.VITE_PORT || '3000'),
      host: '0.0.0.0',
      proxy: {
        '/api': {
          target: env.VITE_API_BASE_URL || 'http://localhost:8081',
          changeOrigin: true
        }
      }
    },
    
    build: {
      outDir: '../internal/pkg/web/dist',
      emptyOutDir: true,
      assetsDir: 'assets',
      rollupOptions: {
        output: {
          manualChunks: {
            'element-plus': ['element-plus'],
            'vue-vendor': ['vue', 'vue-router', 'pinia']
          }
        }
      },
      chunkSizeWarningLimit: 1000
    }
  }
})
