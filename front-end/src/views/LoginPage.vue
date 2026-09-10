<template>
  <div class="home">
    <div class="home__glow" aria-hidden="true" />
    <div class="home__grid" aria-hidden="true" />

    <header class="top">
      <div class="top__brand">
        <DorianMark class="top__mark" />
        <div>
          <p class="top__wordmark">DORIAN</p>
          <p class="top__host">social.dorian.center</p>
        </div>
      </div>
      <span class="top__chip">Admin workspace</span>
    </header>

    <main class="stage">
      <section class="intro">
        <p class="intro__kicker">Social Dorian</p>
        <h1>Sign in to your workspace</h1>
        <p class="intro__lead">
          Manage connected social accounts, proxies, and Gmail mailboxes from one place.
        </p>
        <ul class="features">
          <li>
            <i class="ti ti-share" aria-hidden="true" />
            <div>
              <strong>Social accounts</strong>
              <span>Open remote browsers through each account’s proxy.</span>
            </div>
          </li>
          <li>
            <i class="ti ti-network" aria-hidden="true" />
            <div>
              <strong>Proxy control</strong>
              <span>Track health, exit IPs, and assignments.</span>
            </div>
          </li>
          <li>
            <i class="ti ti-brand-gmail" aria-hidden="true" />
            <div>
              <strong>Mailbox ops</strong>
              <span>Keep recovery mail and PIN codes next to each account.</span>
            </div>
          </li>
        </ul>
      </section>

      <section class="card">
        <h2>Welcome back</h2>
        <p class="card__lead">Use your admin email and password to continue.</p>

        <form class="form" @submit.prevent="submit">
          <label class="field">
            <span>Email</span>
            <input
              v-model.trim="email"
              type="email"
              name="email"
              autocomplete="username"
              placeholder="admin@dorian.center"
              required
            />
          </label>

          <label class="field">
            <span>Password</span>
            <div class="field__password">
              <input
                v-model="password"
                :type="showPassword ? 'text' : 'password'"
                name="password"
                autocomplete="current-password"
                placeholder="Password"
                required
              />
              <button
                class="btn btn-icon field__toggle"
                type="button"
                :aria-label="showPassword ? 'Hide password' : 'Show password'"
                @click="showPassword = !showPassword"
              >
                <i class="ti" :class="showPassword ? 'ti-eye-off' : 'ti-eye'" aria-hidden="true" />
              </button>
            </div>
          </label>

          <p v-if="error" class="form__error" role="alert">{{ error }}</p>

          <button class="btn btn-primary form__submit" type="submit" :disabled="busy">
            <i class="ti" :class="busy ? 'ti-loader-2 spin' : 'ti-login'" aria-hidden="true" />
            {{ busy ? 'Signing in…' : 'Sign in' }}
          </button>
        </form>
      </section>
    </main>

    <footer class="foot">Private workspace · Dorian</footer>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { login, safeNextPath } from '../api/auth'
import DorianMark from '../components/DorianMark.vue'

const route = useRoute()
const router = useRouter()

const email = ref('')
const password = ref('')
const showPassword = ref(false)
const busy = ref(false)
const error = ref('')

onMounted(() => {
  document.title = 'Sign in — Social Dorian'
})

async function submit() {
  error.value = ''
  busy.value = true
  try {
    await login(email.value, password.value)
    document.title = 'Dorian — Social accounts'
    await router.replace(safeNextPath(route.query.next))
  } catch (err) {
    error.value = err.message || 'Could not sign in'
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.home {
  position: relative;
  isolation: isolate;
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  padding: 28px 32px 20px;
  overflow: hidden;
}

.home__glow {
  position: absolute;
  inset: 8% 0 auto;
  height: 55%;
  background:
    radial-gradient(ellipse at 22% 40%, color-mix(in srgb, var(--viper-500) 26%, transparent), transparent 58%),
    radial-gradient(ellipse at 78% 30%, color-mix(in srgb, var(--gold-500) 12%, transparent), transparent 48%);
  pointer-events: none;
  z-index: -1;
}

.home__grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(color-mix(in srgb, var(--hairline) 70%, transparent) 1px, transparent 1px),
    linear-gradient(90deg, color-mix(in srgb, var(--hairline) 70%, transparent) 1px, transparent 1px);
  background-size: 48px 48px;
  mask-image: radial-gradient(circle at 50% 40%, black, transparent 74%);
  opacity: 0.4;
  pointer-events: none;
  z-index: -1;
}

.top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}

.top__brand {
  display: flex;
  align-items: center;
  gap: 12px;
}

.top__mark {
  font-size: 36px;
}

.top__wordmark {
  font-family: var(--mono);
  font-size: 15px;
  font-weight: 600;
  letter-spacing: 0.08em;
}

.top__host {
  margin-top: 1px;
  font-size: 12px;
  color: var(--text-faint);
}

.top__chip {
  padding: 5px 10px;
  border: 0.5px solid var(--hairline);
  border-radius: 999px;
  background: var(--panel);
  color: var(--text-dim);
  font-size: 12px;
}

.stage {
  flex: 1;
  display: grid;
  grid-template-columns: minmax(0, 1.1fr) minmax(320px, 420px);
  gap: 48px;
  align-items: center;
  width: min(1080px, 100%);
  margin: 0 auto;
  padding: 48px 0 32px;
}

.intro__kicker {
  font-family: var(--mono);
  font-size: 12px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--viper-400);
}

.intro h1 {
  margin-top: 8px;
  font-size: clamp(2rem, 4vw, 3rem);
  line-height: 1.1;
}

.intro__lead {
  margin: 14px 0 28px;
  max-width: 38ch;
  font-size: 16px;
  color: var(--text-dim);
}

.features {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.features li {
  display: flex;
  gap: 12px;
  align-items: flex-start;
}

.features .ti {
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  border-radius: 8px;
  background: var(--viper-dim);
  color: var(--viper-400);
  font-size: 16px;
  flex-shrink: 0;
}

.features strong {
  display: block;
  font-size: 14px;
}

.features span {
  font-size: 13px;
  color: var(--text-dim);
}

.card {
  padding: 28px;
  border: 0.5px solid var(--hairline);
  border-radius: 16px;
  background: color-mix(in srgb, var(--panel) 92%, black);
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.35);
  backdrop-filter: blur(10px);
}

.card h2 {
  font-size: 1.35rem;
}

.card__lead {
  margin: 6px 0 22px;
  font-size: 13.5px;
  color: var(--text-dim);
}

.form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-dim);
}

.field__password {
  position: relative;
}

.field__password input {
  padding-right: 40px;
}

.field__toggle {
  position: absolute;
  top: 50%;
  right: 4px;
  transform: translateY(-50%);
}

.form__error {
  margin: 0;
  padding: 8px 10px;
  border-radius: var(--radius);
  border: 0.5px solid color-mix(in srgb, var(--danger) 35%, transparent);
  background: var(--bg-danger);
  color: var(--danger);
  font-size: 13px;
  font-weight: 500;
}

.form__submit {
  width: 100%;
  margin-top: 4px;
  padding: 10px 14px;
}

.foot {
  font-size: 12px;
  color: var(--text-faint);
}

.spin {
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@media (max-width: 860px) {
  .home {
    padding: 20px 16px 16px;
  }

  .stage {
    grid-template-columns: 1fr;
    gap: 28px;
    padding: 28px 0 20px;
  }

  .intro__lead {
    max-width: none;
  }
}
</style>
