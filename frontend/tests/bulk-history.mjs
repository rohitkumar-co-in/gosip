import assert from 'node:assert/strict'
import {readFile} from 'node:fs/promises'
import ts from 'typescript'
const source=await readFile(new URL('../src/utils/bulk-history.ts',import.meta.url),'utf8')
const code=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.ESNext,target:ts.ScriptTarget.ES2022}}).outputText
const {deleteHistoryBatch,collectConversationHistory}=await import('data:text/javascript;base64,'+Buffer.from(code).toString('base64'))
const records=[{kind:'call',id:1},{kind:'sms',id:1},{kind:'sms',id:2}]
const attempts=[],progress=[]
const results=await deleteHistoryBatch([...records,records[0]],'both',async(record,scope)=>{
 attempts.push([record.kind,record.id,scope])
 if(record.kind==='sms'&&record.id===1)throw {response:{data:{error:{message:'Provider failed; local history kept'}}}}
},count=>progress.push(count))
assert.deepEqual(attempts,[['call',1,'both'],['sms',1,'both'],['sms',2,'both']])
assert.deepEqual(progress,[1,2,3])
assert.deepEqual(results.map(item=>item.deleted),[true,false,true])
assert.equal(results[1].error,'Provider failed; local history kept')
const retried=[]
await deleteHistoryBatch(results.filter(item=>!item.deleted).map(item=>item.record),'local',async(record,scope)=>retried.push([record.kind,record.id,scope]))
assert.deepEqual(retried,[['sms',1,'local']])
assert.deepEqual(await deleteHistoryBatch([],'both',async()=>assert.fail('Empty batch must make no request')),[])
console.log('PASS bulk deletion: distinct call/message IDs, provider failure preservation, continued processing, progress, deduplication and retry only failed records')
const pages=[]
const chats=await collectConversationHistory(['+441234567890','+441234567891','+441234567890'],7,async(number,did,offset,limit)=>{
 pages.push([number,did,offset,limit])
 const count=number.endsWith('0')?115:3
 const start=number.endsWith('0')?1:116
 return {data:Array.from({length:Math.min(limit,count-offset)},(_,index)=>({id:start+offset+index,from_number:number,to_number:'+441234567899'})),has_more:offset+limit<count}
})
assert.equal(chats.length,118)
assert.equal(new Set(chats.map(item=>item.id)).size,118)
assert.ok(chats.every(item=>item.kind==='sms'))
assert.deepEqual(pages,[['+441234567890',7,0,100],['+441234567890',7,100,100],['+441234567891',7,0,100]])
await assert.rejects(()=>collectConversationHistory(['+441234567890'],0,async()=>({data:[],has_more:true})),/did not advance/)
await assert.rejects(()=>collectConversationHistory(['+441234567890','+441234567891'],0,async(number)=>{if(number.endsWith('1'))throw new Error('History fetch failed');return {data:[],has_more:false}}),/History fetch failed/)
console.log('PASS conversation selection: different phone numbers, all older pages, business-number scope, duplicate selection and fetch-error handling')
