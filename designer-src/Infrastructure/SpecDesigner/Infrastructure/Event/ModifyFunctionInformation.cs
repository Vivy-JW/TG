using System;
using SpecDesigner.Infrastructure.Model;
using SpecDesignerCommon;

namespace SpecDesigner.Infrastructure.Event
{
	// Token: 0x02000012 RID: 18
	public class ModifyFunctionInformation
	{
		// Token: 0x1700000A RID: 10
		// (get) Token: 0x06000028 RID: 40 RVA: 0x000022BB File Offset: 0x000004BB
		// (set) Token: 0x06000029 RID: 41 RVA: 0x000022C3 File Offset: 0x000004C3
		public Guid ID { get; set; }

		// Token: 0x1700000B RID: 11
		// (get) Token: 0x0600002A RID: 42 RVA: 0x000022CC File Offset: 0x000004CC
		// (set) Token: 0x0600002B RID: 43 RVA: 0x000022D4 File Offset: 0x000004D4
		public PackageKey ProgramKey { get; set; }

		// Token: 0x1700000C RID: 12
		// (get) Token: 0x0600002C RID: 44 RVA: 0x000022DD File Offset: 0x000004DD
		// (set) Token: 0x0600002D RID: 45 RVA: 0x000022E5 File Offset: 0x000004E5
		public string Source { get; set; }

		// Token: 0x1700000D RID: 13
		// (get) Token: 0x0600002E RID: 46 RVA: 0x000022EE File Offset: 0x000004EE
		// (set) Token: 0x0600002F RID: 47 RVA: 0x000022F6 File Offset: 0x000004F6
		public string Modified { get; set; }

		// Token: 0x1700000E RID: 14
		// (get) Token: 0x06000030 RID: 48 RVA: 0x000022FF File Offset: 0x000004FF
		// (set) Token: 0x06000031 RID: 49 RVA: 0x00002307 File Offset: 0x00000507
		public string RemoveContent { get; set; }

		// Token: 0x1700000F RID: 15
		// (get) Token: 0x06000032 RID: 50 RVA: 0x00002310 File Offset: 0x00000510
		// (set) Token: 0x06000033 RID: 51 RVA: 0x00002318 File Offset: 0x00000518
		public string InsertContent { get; set; }

		// Token: 0x17000010 RID: 16
		// (get) Token: 0x06000034 RID: 52 RVA: 0x00002321 File Offset: 0x00000521
		// (set) Token: 0x06000035 RID: 53 RVA: 0x00002329 File Offset: 0x00000529
		public int Offset { get; set; }

		// Token: 0x17000011 RID: 17
		// (get) Token: 0x06000036 RID: 54 RVA: 0x00002332 File Offset: 0x00000532
		// (set) Token: 0x06000037 RID: 55 RVA: 0x0000233A File Offset: 0x0000053A
		public Scope Scope { get; set; }

		// Token: 0x17000012 RID: 18
		// (get) Token: 0x06000038 RID: 56 RVA: 0x00002343 File Offset: 0x00000543
		// (set) Token: 0x06000039 RID: 57 RVA: 0x0000234B File Offset: 0x0000054B
		public string Description { get; set; }

		// Token: 0x17000013 RID: 19
		// (get) Token: 0x0600003A RID: 58 RVA: 0x00002354 File Offset: 0x00000554
		// (set) Token: 0x0600003B RID: 59 RVA: 0x0000235C File Offset: 0x0000055C
		public string Text { get; set; }

		// Token: 0x17000014 RID: 20
		// (get) Token: 0x0600003C RID: 60 RVA: 0x00002365 File Offset: 0x00000565
		// (set) Token: 0x0600003D RID: 61 RVA: 0x0000236D File Offset: 0x0000056D
		public string Content { get; set; }
	}
}
