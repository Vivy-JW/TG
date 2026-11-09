using System;
using SpecDesignerCommon;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.SpecEditor
{
	// Token: 0x02000039 RID: 57
	public class SpecNodeLoadedEventArgs
	{
		// Token: 0x06000180 RID: 384 RVA: 0x0000B0D6 File Offset: 0x000092D6
		public SpecNodeLoadedEventArgs(AbstractSpecNode node, PackageKey programKey)
		{
			this.SpecNode = node;
			this.ProgramKey = programKey;
		}

		// Token: 0x1700002C RID: 44
		// (get) Token: 0x06000181 RID: 385 RVA: 0x0000B0EC File Offset: 0x000092EC
		// (set) Token: 0x06000182 RID: 386 RVA: 0x0000B0F4 File Offset: 0x000092F4
		public AbstractSpecNode SpecNode { get; private set; }

		// Token: 0x1700002D RID: 45
		// (get) Token: 0x06000183 RID: 387 RVA: 0x0000B0FD File Offset: 0x000092FD
		// (set) Token: 0x06000184 RID: 388 RVA: 0x0000B105 File Offset: 0x00009305
		public PackageKey ProgramKey { get; private set; }
	}
}
