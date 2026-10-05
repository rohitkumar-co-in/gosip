<script setup lang="ts">
import {ref,onMounted} from 'vue'
import {RouterLink} from 'vue-router'
import api from '@/api/client'
const numbers=ref<{sid:string;phone_number:string;friendly_name:string;capabilities:{voice:boolean;sms:boolean}}[]>([]),accounts=ref<{number:string;username:string;state:string}[]>([]),error=ref(''),loading=ref(false)
async function load(){loading.value=true;error.value='';try{const [n,a]=await Promise.all([api.get('/business/numbers'),api.get('/business/accounts')]);numbers.value=n.data.data;accounts.value=a.data.data}catch{error.value='Could not fetch owned Twilio numbers'}finally{loading.value=false}}
function assigned(n:string){return accounts.value.find(a=>a.number===n)}
onMounted(load)
</script>
<template><div class="space-y-6"><div class="flex justify-between"><div><h1 class="text-2xl font-semibold">Twilio numbers</h1><p class="text-sm text-gray-500 mt-2">Live inventory of numbers already owned by your Twilio account.</p></div><button @click="load" :disabled="loading" class="border rounded px-4 py-2">Refresh from Twilio</button></div><p v-if="error" class="text-red-600">{{error}}</p><div class="bg-white dark:bg-gray-800 rounded-lg overflow-x-auto"><table class="min-w-full text-left text-sm"><thead><tr><th class="p-4">Number</th><th class="p-4">Voice / SMS</th><th class="p-4">Assigned user</th></tr></thead><tbody><tr v-for="n in numbers" :key="n.sid" class="border-t"><td class="p-4">{{n.phone_number}}<p class="text-gray-500">{{n.friendly_name}}</p></td><td class="p-4">{{n.capabilities.voice?'Voice':'No voice'}} · {{n.capabilities.sms?'SMS':'No SMS'}}</td><td class="p-4">{{assigned(n.phone_number)?.username||'Available'}}<p class="text-gray-500">{{assigned(n.phone_number)?.state}}</p></td></tr></tbody></table></div><RouterLink to="/devices" class="text-primary">Assign a number from SIP Users →</RouterLink></div></template>
