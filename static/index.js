const { createApp } = Vue

const app = createApp({
    data() {
        return { dark: false, current: null, clients: [] };
    },
    async mounted() {
        this.media();
        this.fetch();
        this.subscribe();
    },
    methods: {
        media() {
            const media = window.matchMedia("(prefers-color-scheme: dark)");
            this.dark = media.matches;
            media.onchange = (ev) => (this.dark = ev.matches);
        },

        async fetch() {
            const res = await fetch("/api/state");
            const { current, clients } = await res.json();
            this.current = current;
            this.clients = clients;
        },
        subscribe() {
            const sse = new EventSource("p2p/sse");
            sse.onmessage = (e) => {
                const { current, clients } = JSON.parse(e.data);
                this.current = current;
                this.clients = clients;
            };
        },
        date(v) {
            return new Date(v).toLocaleString();
        },
    },

    computed: {
        classes() {
            return {
                main: { "main-dark": this.dark },
                item: { "text-bg-dark": this.dark },
                badge: { "bg-primary": !this.dark, "text-bg-light": this.dark },
            };
        },
    },
})

app.component("date-since", {
    props: ["date"],
    mounted() {
        this.reset();
    },
    destroyed() {
        clearInterval(this.interval);
    },
    methods: {
        reset() {
            this.interval = setInterval(() => (this.display = moment(this.date).fromNow()), 1000);
        },
    },
    data() {
        return { display: "", interval: null };
    },
    watch: {
        date() {
            this.reset();
        },
    },
    template: `{{ display }}`,
});





app.mount("#app");
