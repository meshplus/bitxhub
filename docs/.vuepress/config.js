module.exports = {
    title: 'BitXHub文档',
    description: 'BitXHub部署文档、使用文档、设计文档站',
    base: '/bitxhub/',
    themeConfig: {
        logo: '/images/logo.png',
        nav: [
            {text: '官网', link: 'https://bitxhub.hyperchain.cn'},
            {text: 'Github', link: 'https://github.com/meshplus/bitxhub'},
        ],

        sidebar: [
            {
                title: '快速开始',
                path: '/quick/',
            },
            {
                title: '部署文档',
                path: '/deploy/',
            },
            {
                title: '使用文档',
                path: '/usage/',
            },
            {
                title: '开发文档',
                path: '/develop/',
                children: [
                    {
                        title: '共识算法插件化使用文档',
                        path: '/develop/consensus_usage'
                    },
                    {
                        title: '共识算法插件化设计文档',
                        path: '/develop/consensus_design'
                    },
                    {
                        title: '跨链合约编写规范',
                        path: '/develop/interchain_contract'
                    },
                    {
                        title: '验证引擎规则编写规范',
                        path: '/develop/rule'
                    }
                ],
            },
            {
                title: 'FAQ',
                path: '/faq/',
            },
        ]

    }
}
