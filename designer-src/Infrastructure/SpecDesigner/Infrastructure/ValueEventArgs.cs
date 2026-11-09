using System;

namespace SpecDesigner.Infrastructure
{
	// Token: 0x0200001F RID: 31
	public class ValueEventArgs : EventArgs
	{
		// Token: 0x17000019 RID: 25
		// (get) Token: 0x06000068 RID: 104 RVA: 0x00002A2E File Offset: 0x00000C2E
		// (set) Token: 0x06000069 RID: 105 RVA: 0x00002A36 File Offset: 0x00000C36
		public string Content { get; private set; }

		// Token: 0x0600006A RID: 106 RVA: 0x00002A3F File Offset: 0x00000C3F
		public ValueEventArgs(string content)
		{
			this.Content = content;
		}
	}
}
