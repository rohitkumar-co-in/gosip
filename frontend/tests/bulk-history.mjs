import assert from 'node:assert/strict'
import {readFile} from 'node:fs/promises'
import ts from 'typescript'
const source=await readFile(new URL('../src/utils/bulk-history.ts',import.meta.url),'utf8')
const code=ts.transpileModule(source,{compilerOptions:{module:ts.ModuleKind.ESNext,target:ts.ScriptTarget.ES2022}}).outputText
const {deleteHistoryBatch}=await import('data:text/javascript;base64,'+Buffer.from(code).toString('base64'))
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
