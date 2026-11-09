using System;

namespace SpecDesigner.Infrastructure
{
	// Token: 0x0200001D RID: 29
	public class ElementEventArgs : EventArgs
	{
		// Token: 0x17000017 RID: 23
		// (get) Token: 0x06000061 RID: 97 RVA: 0x000029DC File Offset: 0x00000BDC
		// (set) Token: 0x06000062 RID: 98 RVA: 0x000029E4 File Offset: 0x00000BE4
		public string Type { get; private set; }

		// Token: 0x06000063 RID: 99 RVA: 0x000029ED File Offset: 0x00000BED
		public ElementEventArgs(string type)
		{
			this.Type = type;
		}
	}
}
