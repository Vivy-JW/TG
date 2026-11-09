using System;

namespace SpecDesignerCommon
{
	// Token: 0x02000003 RID: 3
	public class ComponentDeletedArgs
	{
		// Token: 0x17000008 RID: 8
		// (get) Token: 0x06000026 RID: 38 RVA: 0x00002621 File Offset: 0x00000821
		// (set) Token: 0x06000027 RID: 39 RVA: 0x00002629 File Offset: 0x00000829
		public string Name { get; set; }

		// Token: 0x17000009 RID: 9
		// (get) Token: 0x06000028 RID: 40 RVA: 0x00002632 File Offset: 0x00000832
		// (set) Token: 0x06000029 RID: 41 RVA: 0x0000263A File Offset: 0x0000083A
		public string ParentName { get; set; }

		// Token: 0x0600002A RID: 42 RVA: 0x00002643 File Offset: 0x00000843
		public ComponentDeletedArgs(string name, string parent)
		{
			this.Name = name;
			this.ParentName = parent;
		}
	}
}
