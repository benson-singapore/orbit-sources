package main

// Auto-generated catalog from interviewGuide _sidebar.md menu.

type catalogItem struct {
	Path  string // markdown path relative to site root, e.g. docs/JavaBasic.md
	Title string
}

type catalogSection struct {
	ID    string
	Label string
	Items []catalogItem
}

var catalogSections = []catalogSection{
	{ID: "home", Label: "首页", Items: []catalogItem{
		{Path: "README.md", Title: "《大厂面试指北》首页"},
	}},
	{ID: "java", Label: "Java", Items: []catalogItem{
		{Path: "docs/JavaBasic.md", Title: "基础"},
		{Path: "docs/ArrayList.md", Title: "ArrayList和LinkedList"},
		{Path: "docs/HashMap.md", Title: "HashMap和ConcurrentHashMap"},
		{Path: "docs/JavaMultiThread.md", Title: "多线程"},
		{Path: "docs/Lock.md", Title: "锁相关"},
	}},
	{ID: "redis", Label: "Redis", Items: []catalogItem{
		{Path: "docs/RedisBasic.md", Title: "基础"},
		{Path: "docs/RedisDataStruct.md", Title: "数据结构"},
		{Path: "docs/RedisStore.md", Title: "持久化(AOF和RDB)"},
		{Path: "docs/RedisUserful.md", Title: "高可用(主从切换和哨兵机制)"},
	}},
	{ID: "mysql", Label: "MySQL", Items: []catalogItem{
		{Path: "docs/MySQLNote.md", Title: "基础"},
		{Path: "docs/MySQLWork.md", Title: "慢查询优化实践"},
	}},
	{ID: "jvm", Label: "JVM", Items: []catalogItem{
		{Path: "docs/JavaJVM.md", Title: "基础"},
	}},
	{ID: "kafka", Label: "Kafka", Items: []catalogItem{
		{Path: "docs/Kafka.md", Title: "Kafka"},
	}},
	{ID: "zookeeper", Label: "ZooKeeper", Items: []catalogItem{
		{Path: "docs/ZooKeeper.md", Title: "ZooKeeper"},
	}},
	{ID: "http", Label: "HTTP", Items: []catalogItem{
		{Path: "docs/HTTP.md", Title: "HTTP"},
	}},
	{ID: "spring", Label: "Spring", Items: []catalogItem{
		{Path: "docs/Spring.md", Title: "Spring"},
	}},
	{ID: "nginx", Label: "Nginx", Items: []catalogItem{
		{Path: "docs/Nginx.md", Title: "Nginx"},
	}},
	{ID: "system-design", Label: "系统设计", Items: []catalogItem{
		{Path: "docs/SystemDesign.md", Title: "系统设计"},
	}},
	{ID: "algorithm", Label: "算法", Items: []catalogItem{
		{Path: "docs/CodingInterviews.md", Title: "《剑指Offer》解题思考"},
		{Path: "docs/LeetCode.md", Title: "《LeetCode热门100题》解题思考(上)"},
		{Path: "docs/LeetCode1.md", Title: "《LeetCode热门100题》解题思考(下)"},
	}},
	{ID: "bat", Label: "大厂面试公众号文章系列", Items: []catalogItem{
		{Path: "docs/BATInterview.md", Title: "大厂面试公众号文章系列"},
	}},
	{ID: "notes", Label: "读书笔记", Items: []catalogItem{
		{Path: "docs/RedisBook1.md", Title: "《Redis设计与实现》读书笔记 上"},
		{Path: "docs/RedisBook2.md", Title: "《Redis设计与实现》读书笔记 下"},
		{Path: "docs/MySQLBook1.md", Title: "《MySQL必知必会》读书笔记"},
		{Path: "docs/JVMBook.md", Title: "《深入理解Java虚拟机-第三版》读书笔记"},
	}},
	{ID: "books", Label: "好书推荐", Items: []catalogItem{
		{Path: "docs/bookRecommend.md", Title: "好书推荐"},
	}},
}

var catalogByID = map[string]*catalogSection{}

func init() {
	for i := range catalogSections {
		catalogByID[catalogSections[i].ID] = &catalogSections[i]
	}
}
