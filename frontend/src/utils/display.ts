export function list<T>(value:unknown):T[]{return Array.isArray(value)?value as T[]:[]}
export function dateTime(value?:string):string{
 if(!value)return '—'
 const normalized=/[zZ]|[+-]\d\d:\d\d$/.test(value)?value:value.replace(' ','T')+'Z'
 const date=new Date(normalized)
 return Number.isNaN(date.getTime())?'—':date.toLocaleString()
}
