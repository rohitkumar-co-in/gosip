<script setup lang="ts">
import {ref} from 'vue'
import api from '@/api/client'
import ConsoleDialog from './ConsoleDialog.vue'
const props=defineProps<{kind:'call'|'sms';id:number;from:string;to:string}>()
const emit=defineEmits<{close:[];deleted:[]}>()
const scope=ref<'local'|'both'>('local'),confirmed=ref(false),busy=ref(false),error=ref('')
async function remove(){
 if(!confirmed.value||busy.value)return
 busy.value=true;error.value=''
 try{await api.delete(`/business/activity/${props.kind}/${props.id}`,{data:{scope:scope.value,confirm:true}});emit('deleted')}
 catch(err:unknown){error.value=(err as {response?:{data?:{error?:{message?:string}}}}).response?.data?.error?.message||'Deletion failed. Refresh history before retrying.'}
 finally{busy.value=false}
}
</script>
<template><ConsoleDialog :label="kind==='sms'?'Delete message':'Delete call history'" @close="!busy&&emit('close')"><form class="space-y-5" @submit.prevent="remove">
 <h2 class="text-xl">{{kind==='sms'?'Delete message':'Delete call history'}}?</h2>
 <p class="text-sm break-words">{{from}} → {{to}}</p>
 <label class="block text-sm">Delete from<select v-model="scope" :disabled="busy" class="block w-full border rounded p-3 mt-2" @change="confirmed=false"><option value="local">Dashboard only</option><option value="both">Dashboard and Twilio</option></select></label>
 <p class="text-sm">{{scope==='both'?'This permanently deletes the selected record from your dashboard and Twilio logs.':'This removes the record from your dashboard. The Twilio record remains and can be imported again.'}}</p>
 <p v-if="scope==='both'" class="text-sm">{{kind==='sms'?'Twilio also deletes associated message media. This does not remove messages already delivered to phones.':'Call recordings, transcriptions and other call legs are kept. Only this call record is deleted.'}}</p>
 <p class="text-sm">Existing backups are kept. This action cannot be undone from the dashboard.</p>
 <label class="flex gap-3 items-start text-sm"><input v-model="confirmed" type="checkbox" :disabled="busy" class="mt-1" />I confirm deletion of this record.</label>
 <p v-if="error" role="alert" class="text-red-600 text-sm">{{error}}</p>
 <div class="dialog-actions"><button type="button" class="border rounded px-4 py-2" :disabled="busy" @click="emit('close')">Cancel</button><button type="submit" class="bg-red-700 text-white rounded px-4 py-2" :disabled="!confirmed||busy">{{busy?'Deleting…':'Delete record'}}</button></div>
</form></ConsoleDialog></template>
