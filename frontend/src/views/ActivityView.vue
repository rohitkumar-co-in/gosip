<script setup lang="ts">
import {ref,onMounted,watch} from 'vue'
import {useRoute} from 'vue-router'
import api from '@/api/client'
import {dateTime} from '@/utils/display'
const route=useRoute()
interface Activity {kind:string;id:number;direction:string;from_number:string;to_number:string;status:string;occurred_at:string;duration:number;body:string;actor:string}
const data=ref<Activity[]>([]),loading=ref(false),error=ref(''),offset=ref(0),kind=ref(route.path==='/calls'?'call':'')
async function load(){loading.value=true;error.value='';try{data.value=(await api.get('/business/activity',{params:{offset:offset.value,kind:kind.value}})).data.data}catch{error.value='Activity could not be loaded. Try again.'}finally{loading.value=false}}

watch(()=>route.path,()=>{kind.value=route.path==='/calls'?'call':'';offset.value=0;load()})
onMounted(load)
</script>
<template><div class="space-y-6 page-enter">
 <div class="flex flex-wrap items-start justify-between gap-4"><div><p class="text-primary text-xs tracking-widest uppercase mb-2">Leadomi SIP · Operations</p><h1 class="text-3xl">{{route.path==='/calls'?'Call history':'Activity log'}}</h1><p class="text-gray-500 text-sm mt-2">Calls and SMS, with direction, numbers, account, status, and time.</p></div><button class="border rounded-lg px-4 py-2" @click="load" :disabled="loading">Refresh</button></div>
 <div class="glass rounded-lg p-4 flex flex-wrap gap-4 items-center"><label>Type <select v-model="kind" @change="offset=0;load()" class="ml-2 border rounded p-2"><option value="">Calls and SMS</option><option value="call">Calls</option><option value="sms">SMS</option></select></label><p class="text-sm text-gray-500">Times use your device’s timezone. “Assigned to” identifies the receiving account, not proof someone answered or read it.</p></div>
 <p v-if="error" role="alert" class="text-red-600">{{error}}</p>
 <div class="glass rounded-lg overflow-x-auto"><table class="responsive-table w-full text-sm text-left"><thead><tr><th class="p-4">Time / type</th><th class="p-4">Who / direction</th><th class="p-4">From → To</th><th class="p-4">Status / details</th></tr></thead><tbody><tr v-for="item in data" :key="`${item.kind}-${item.id}`" class="border-t"><td data-label="Time / type" class="p-4 whitespace-nowrap">{{dateTime(item.occurred_at)}}<p class="text-primary uppercase text-xs mt-1">{{item.kind}}</p></td><td data-label="Who / direction" class="p-4">{{item.actor}}<p class="text-gray-500">{{item.direction}}</p></td><td data-label="From → To" class="p-4 font-mono whitespace-nowrap">{{item.from_number}}<p class="text-gray-500 mt-1">→ {{item.to_number}}</p></td><td data-label="Status / details" class="p-4"><span>{{item.status}}</span><p v-if="item.kind==='call'" class="text-gray-500">{{item.duration}} seconds</p><p v-else class="text-gray-500 whitespace-pre-wrap break-words max-w-md">{{item.body}}</p></td></tr><tr v-if="!data.length"><td colspan="4" class="p-8 text-gray-500">{{loading?'Loading activity…':'No activity on this page.'}}</td></tr></tbody></table></div>
 <div class="flex flex-wrap gap-3 items-center justify-between"><button class="border px-4 py-2 rounded" :disabled="!offset||loading" @click="offset=Math.max(0,offset-50);load()">Previous</button><span class="text-sm text-gray-500">{{data.length?'Records '+(offset+1)+'–'+(offset+data.length):'No records'}}</span><button class="border px-4 py-2 rounded" :disabled="data.length<50||loading" @click="offset+=50;load()">Next</button></div>
</div></template>
