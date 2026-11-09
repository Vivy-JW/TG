using System;
using System.Xml.Serialization;
using SpecDesignerCommon.Helpers;

namespace SpecDesignerCommon
{
	// Token: 0x0200002D RID: 45
	public class FJSObject
	{
		// Token: 0x0400008B RID: 139
		[XmlAttribute]
		public ComponentType Type;

		// Token: 0x0400008C RID: 140
		[XmlText]
		public string Text;
	}
}
