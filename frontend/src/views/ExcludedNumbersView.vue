<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '@/api/client'
import ConsoleDialog from '@/components/ConsoleDialog.vue'
import { dateTime } from '@/utils/display'
interface Exclusion { number:string;reason:string;created_at:string }
const entries=ref<Exclusion[]>([]),number=ref(''),reason=ref(''),busy=ref(false),error=ref(''),notice=ref(''),removing=ref<Exclusion|null>(null)
function failure(e:unknown){return (e as {response?:{data?:{error?:{message?:string}}}}).response?.data?.error?.message||'Could not save number protection. Refresh before retrying.'}
async function load(){busy.value=true;error.value='';try{entries.value=(await api.get('/business/excluded-numbers')).data.data}catch(e){error.value=failure(e)}finally{busy.value=false}}
async function protect(){busy.value=true;error.value='';notice.value='';try{entries.value=(await api.put('/business/excluded-numbers',{number:number.value.trim(),reason:reason.value})).data.data;notice.value='Number excluded. Existing calls, SMS and connections were preserved.';number.value='';reason.value=''}catch(e){error.value=failure(e)}finally{busy.value=false}}
async function unprotect(){if(!removing.value)return;busy.value=true;error.value='';notice.value='';try{entries.value=(await api.delete('/business/excluded-numbers',{params:{number:removing.value.number}})).data.data;removing.value=null;notice.value='Exclusion removed. No connections were changed.'}catch(e){error.value=failure(e)}finally{busy.value=false}}
onMounted(load)
</script>
<template>
 <div class="space-y-6">
  <div class="flex flex-wrap justify-between gap-4"><div><h1 class="text-2xl font-semibold">Excluded numbers</h1><p class="mt-2 text-sm text-gray-500">Protect numbers used by another system from configuration changes.</p></div><button @click="load" :disabled="busy" class="border rounded px-4 py-2">Refresh</button></div>
  <p class="settings-card text-sm">Excluded numbers cannot be assigned, reconfigured or forced through connection review. Linked SIP-user edits, disabling/deletion and local routing changes are blocked. Existing calls, SMS, webhooks and phone connections continue unchanged. Remove an exclusion explicitly before configuring that number. This protects changes made through Leadomi SIP; changes directly in Twilio remain under your control.</p>
  <p v-if="error" role="alert" class="rounded bg-red-50 text-red-800 p-4">{{error}}</p><p v-if="notice" role="status" class="text-green-600">{{notice}}</p>
  <form @submit.prevent="protect" class="settings-card space-y-4"><h2 class="font-semibold">Exclude a number</h2><label class="block">Phone number<input v-model="number" required pattern="\+[1-9][0-9]{7,14}" placeholder="+447…" class="block w-full border rounded p-2 mt-1" /></label><label class="block">Reason (optional)<input v-model="reason" maxlength="256" placeholder="Used by another service" class="block w-full border rounded p-2 mt-1" /></label><button :disabled="busy" class="bg-primary text-white rounded px-4 py-2">Exclude number</button></form>
  <div class="settings-card overflow-x-auto"><table class="w-full text-left text-sm"><thead><tr><th class="p-3">Number</th><th class="p-3">Reason</th><th class="p-3">Protected since</th><th class="p-3">Actions</th></tr></thead><tbody><tr v-for="entry in entries" :key="entry.number" class="border-t"><td class="p-3">{{entry.number}}</td><td class="p-3">{{entry.reason||'—'}}</td><td class="p-3">{{dateTime(entry.created_at)}}</td><td class="p-3"><button @click="removing=entry" :disabled="busy" class="text-primary">Remove exclusion</button></td></tr><tr v-if="!entries.length"><td colspan="4" class="p-3 text-gray-500">No excluded numbers. Add only the numbers you want to protect.</td></tr></tbody></table></div>
  <ConsoleDialog v-if="removing" label="Remove number exclusion" @close="!busy&&(removing=null)"><div class="space-y-4"><h2 class="text-xl">Remove protection for {{removing.number}}?</h2><p>This allows future assignment and configuration changes. Removing the exclusion does not change its current connections. Assigning it later still requires connection review and explicit confirmation.</p><div class="dialog-actions"><button @click="removing=null" :disabled="busy" class="border rounded px-4 py-2">Cancel</button><button @click="unprotect" :disabled="busy" class="bg-primary text-white rounded px-4 py-2">Remove exclusion</button></div></div></ConsoleDialog>
 </div>
</template>
