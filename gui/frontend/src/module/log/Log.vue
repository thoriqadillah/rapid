<script setup lang="ts">
import Header from '@/components/Header.vue';
import Cato from '@/assets/images/cato.svg'
import { computed, ref, reactive, watch } from 'vue';
import { useRouteQuery } from '@vueuse/router';
import useLog from './api'

const now = new Date()
const date = reactive({
    day: now.getDate() < 10 ? `0${now.getDate()}` : now.getDate(),
    month: now.getMonth()+1 < 10 ? `0${now.getMonth()+1}` : now.getMonth()+1,
    year: now.getFullYear()
})

const log = useLog()
const search = useRouteQuery('search', '')
const logs = ref<string[]>([])

async function init() {
    logs.value = await log.get(`${date.day}-${date.month}-${date.year}`)
}

init()

watch(date, async (val) => {
    logs.value = await log.get(`${date.day}-${date.month}-${date.year}`)
})

const items = computed(() => {
    if (search.value) {
        return logs.value.filter(log => log.toLowerCase().includes(search.value.toLowerCase()))
    }

    return logs.value
})

</script>

<template>
    <Header />
    <div :class="`${!items || items.length === 0 ? '' : 'bg-secondary border border-muted mb-3 rounded-md p-2'} mt-7 h-full`">
        <div v-if="items.length === 0" class="w-fit mx-auto">
            <img :src="Cato" alt="empty" class="mx-auto my-auto w-[20rem] h-[80vh]">
        </div>
        <div v-else class="w-fit flex flex-col gap-2 py-1">
            <div v-for="(item, i) in items" :key="i" :class="`flex gap-2 ${item.toLowerCase().includes('error') ? 'text-destructive' : ''}`">
                <span class="text-right w-[2rem] [&:not(:first-child)]:mt-0 text-xs font-mono border-r pr-2 border-muted">{{ i+1 }}</span>
                <span class="text-left [&:not(:first-child)]:mt-0 text-xs xl:w-[75rem] font-mono">{{ item }}</span>
            </div>
        </div>
    </div>
</template>