<script setup lang="ts">
import {ref} from 'vue'
import api from '@/api/client'
import ConsoleDialog from './ConsoleDialog.vue'
import {deleteHistoryBatch,type HistoryRecord,type HistoryScope,type HistoryResult} from '@/utils/bulk-history'
const props=defineProps<{records:HistoryRecord[];conversations?:string[];businessNumber?:string}>()
const emit=defineEmits<{close:[];changed:[]}>()
const pending=ref([...props.records]),scope=ref<HistoryScope>('local'),confirmed=ref(false),busy=ref(false),processed=ref(0),total=ref(0),deleted=ref(0),failures=ref<HistoryResult[]>([])
async function remove(){
 if(!confirmed.value||busy.value||!pending.value.length)return
 busy.value=true;processed.value=0;total.value=pending.value.length;failures.value=[]
 const result=await deleteHistoryBatch(pending.value,scope.value,(record,target)=>api.delete(`/business/activity/${record.kind}/${record.id}`,{data:{scope:target,confirm:true}}),count=>processed.value=count)
 failures.value=result.filter(item=>!item.deleted)
 pending.value=failures.value.map(item=>item.record)
 deleted.value+=result.filter(item=>item.deleted).length
 busy.value=false;confirmed.value=false
 if(result.some(item=>item.deleted))emit('changed')
 if(!pending.value.length)emit('close')
}
</script>
<template><ConsoleDialog label="Delete selected history" @close="!busy&&emit('close')"><form class="space-y-5" @submit.prevent="remove">
 <h2 class="text-xl">{{conversations?.length?'Delete '+conversations.length+' conversations?':'Delete '+pending.length+' selected records?'}}</h2>
 <div v-if="conversations?.length" class="space-y-2 text-sm"><p>{{pending.length}} messages, including older history. Business number: {{businessNumber}}.</p><ul class="max-h-32 overflow-y-auto"><li v-for="number in conversations" :key="number">{{number}}</li></ul><p>Only the prepared messages in these conversations will be processed. Messages arriving afterward are kept.</p></div>
 <p v-else class="text-sm">Only the selected messages and call records will be processed.</p>
 <label class="block text-sm">Delete from<select v-model="scope" :disabled="busy" class="block w-full border rounded p-3 mt-2" @change="confirmed=false"><option value="local">Dashboard only</option><option value="both">Dashboard and Twilio</option></select></label>
 <p class="text-sm">{{scope==='both'?'Selected records will be permanently removed from the dashboard and Twilio logs.':'Selected records will be removed from the dashboard. Twilio records remain and can be imported again.'}}</p>
 <p v-if="scope==='both'" class="text-sm">Message media is deleted with its Twilio message. Delivered phone messages, call recordings, transcriptions and other call legs are kept.</p>
 <p class="text-sm">Active or failed records are kept and reported individually. Existing backups remain unchanged.</p>
 <label class="flex gap-3 items-start text-sm"><input v-model="confirmed" type="checkbox" :disabled="busy" class="mt-1" />I confirm deletion of these selected records.</label>
 <p v-if="busy" role="status" class="text-sm">Processing {{processed}} of {{total}}… Keep this page open until processing finishes.</p>
 <div v-if="failures.length" role="alert" class="space-y-2 text-sm"><p>{{deleted}} deleted; {{failures.length}} kept. Review the failures before retrying.</p><ul class="max-h-48 overflow-y-auto space-y-2"><li v-for="item in failures" :key="`${item.record.kind}-${item.record.id}`" class="break-words">{{item.record.kind==='sms'?'Message':'Call'}} #{{item.record.id}}: {{item.error}}</li></ul></div>
 <div class="dialog-actions"><button type="button" class="border rounded px-4 py-2" :disabled="busy" @click="emit('close')">{{failures.length?'Close':'Cancel'}}</button><button type="submit" class="bg-red-700 text-white rounded px-4 py-2" :disabled="!confirmed||busy">{{busy?'Deleting…':failures.length?'Retry kept records':'Delete selected'}}</button></div>
</form></ConsoleDialog></template>
