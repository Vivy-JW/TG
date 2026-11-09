using System;

namespace SpecDesigner.Infrastructure.Event
{
	// Token: 0x0200000D RID: 13
	public class CreateFunctionInformation
	{
		// Token: 0x17000008 RID: 8
		// (get) Token: 0x0600001F RID: 31 RVA: 0x00002263 File Offset: 0x00000463
		// (set) Token: 0x06000020 RID: 32 RVA: 0x0000226B File Offset: 0x0000046B
		public string Content { get; private set; }

		// Token: 0x17000009 RID: 9
		// (get) Token: 0x06000021 RID: 33 RVA: 0x00002274 File Offset: 0x00000474
		// (set) Token: 0x06000022 RID: 34 RVA: 0x0000227C File Offset: 0x0000047C
		public int Offset { get; private set; }

		// Token: 0x06000023 RID: 35 RVA: 0x00002285 File Offset: 0x00000485
		public CreateFunctionInformation(string content, int startOffset)
		{
			this.Content = content;
			this.Offset = startOffset;
		}
	}
}
