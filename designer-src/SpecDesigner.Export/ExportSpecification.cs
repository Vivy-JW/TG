using System;
using SpecDesigner.Export.Adapters;

namespace SpecDesigner.Export
{
	// Token: 0x02000005 RID: 5
	public class ExportSpecification
	{
		// Token: 0x06000033 RID: 51 RVA: 0x000033F6 File Offset: 0x000015F6
		public ExportSpecification(IExportAdapter adapter)
		{
			this._adapter = adapter;
		}

		// Token: 0x06000034 RID: 52 RVA: 0x00003405 File Offset: 0x00001605
		public void Export(string templateFilePath)
		{
			this._adapter.Export(templateFilePath);
		}

		// Token: 0x0400000B RID: 11
		private IExportAdapter _adapter;
	}
}
