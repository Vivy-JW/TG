using System;

namespace SpecDesigner.CodeEditWindow.View
{
	// Token: 0x0200000C RID: 12
	public class ErrorList
	{
		// Token: 0x17000008 RID: 8
		// (get) Token: 0x06000061 RID: 97 RVA: 0x000055A8 File Offset: 0x000037A8
		// (set) Token: 0x06000060 RID: 96 RVA: 0x0000559F File Offset: 0x0000379F
		public bool check { get; set; }

		// Token: 0x17000009 RID: 9
		// (get) Token: 0x06000063 RID: 99 RVA: 0x000055B9 File Offset: 0x000037B9
		// (set) Token: 0x06000062 RID: 98 RVA: 0x000055B0 File Offset: 0x000037B0
		public string head { get; set; }

		// Token: 0x1700000A RID: 10
		// (get) Token: 0x06000065 RID: 101 RVA: 0x000055CA File Offset: 0x000037CA
		// (set) Token: 0x06000064 RID: 100 RVA: 0x000055C1 File Offset: 0x000037C1
		public int h_line { get; set; }

		// Token: 0x1700000B RID: 11
		// (get) Token: 0x06000067 RID: 103 RVA: 0x000055DB File Offset: 0x000037DB
		// (set) Token: 0x06000066 RID: 102 RVA: 0x000055D2 File Offset: 0x000037D2
		public int h_blankspace { get; set; }

		// Token: 0x1700000C RID: 12
		// (get) Token: 0x06000069 RID: 105 RVA: 0x000055EC File Offset: 0x000037EC
		// (set) Token: 0x06000068 RID: 104 RVA: 0x000055E3 File Offset: 0x000037E3
		public string tail { get; set; }

		// Token: 0x1700000D RID: 13
		// (get) Token: 0x0600006B RID: 107 RVA: 0x000055FD File Offset: 0x000037FD
		// (set) Token: 0x0600006A RID: 106 RVA: 0x000055F4 File Offset: 0x000037F4
		public int t_line { get; set; }

		// Token: 0x1700000E RID: 14
		// (get) Token: 0x0600006D RID: 109 RVA: 0x0000560E File Offset: 0x0000380E
		// (set) Token: 0x0600006C RID: 108 RVA: 0x00005605 File Offset: 0x00003805
		public int t_blankspace { get; set; }
	}
}
