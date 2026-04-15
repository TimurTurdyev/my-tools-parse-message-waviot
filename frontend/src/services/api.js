// Изолирующий слой над Wails-биндингами.
// Компоненты Vue должны дёргать только эти функции, а не
// `window.go.main.App.*` или прямые импорты из wailsjs/*.

import {
  AuthStatus,
  ClearJWT,
  GetMessages,
  OpenLoginInBrowser,
  SetJWTManually,
  StartJWTCapture,
  StopJWTCapture,
} from '../../wailsjs/go/main/App.js'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime.js'

export function fetchAuthStatus() {
  return AuthStatus()
}

export function openLoginInBrowser() {
  return OpenLoginInBrowser()
}

export function setJwtManually(raw) {
  return SetJWTManually(raw)
}

export function clearJwt() {
  return ClearJWT()
}

export function fetchMessages(url) {
  if (!url) return Promise.reject(new Error('URL пуст'))
  return GetMessages(url)
}

export function startJwtCapture() {
  return StartJWTCapture()
}

export function stopJwtCapture() {
  return StopJWTCapture()
}

export function onJwtReceived(cb) {
  EventsOn('jwt:received', cb)
  return () => EventsOff('jwt:received')
}
