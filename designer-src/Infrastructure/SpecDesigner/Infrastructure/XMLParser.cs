using System;
using System.Collections.Generic;
using System.IO;
using System.Xml;
using System.Xml.Linq;
using SpecDesigner.Infrastructure.Model;

namespace SpecDesigner.Infrastructure
{
	// Token: 0x0200001C RID: 28
	public class XMLParser
	{
		// Token: 0x0600004E RID: 78 RVA: 0x000023FC File Offset: 0x000005FC
		public void Parse(string xml)
		{
			if (string.IsNullOrEmpty(xml))
			{
				return;
			}
			using (XmlReader xmlReader = XmlReader.Create(new StringReader(xml)))
			{
				if (this.ParseStart != null)
				{
					this.ParseStart(this, new EventArgs());
				}
				while (xmlReader.Read())
				{
					XmlNodeType nodeType = xmlReader.NodeType;
					switch (nodeType)
					{
					case XmlNodeType.Element:
						if (this.ElementStart != null)
						{
							this.ElementStart(this, new ElementEventArgs(xmlReader.Name));
						}
						if (xmlReader.HasAttributes)
						{
							AttributesEventArgs e = new AttributesEventArgs();
							for (int i = 0; i < xmlReader.AttributeCount; i++)
							{
								xmlReader.MoveToAttribute(i);
								e.AddAttribute(xmlReader.Name, xmlReader.GetAttribute(i));
							}
							if (this.AttributesGet != null)
							{
								this.AttributesGet(this, e);
							}
						}
						xmlReader.MoveToElement();
						if (xmlReader.IsEmptyElement && this.ElementEnd != null)
						{
							this.ElementEnd(this, null);
						}
						break;
					case XmlNodeType.Attribute:
						break;
					case XmlNodeType.Text:
						if (this.ValueGet != null)
						{
							this.ValueGet(this, new ValueEventArgs(xmlReader.Value));
						}
						break;
					case XmlNodeType.CDATA:
						if (this.CDataGet != null)
						{
							this.CDataGet(this, new CDataEventArgs(xmlReader.Value));
						}
						break;
					default:
						switch (nodeType)
						{
						case XmlNodeType.EndElement:
							if (this.ElementEnd != null)
							{
								this.ElementEnd(this, null);
							}
							break;
						case XmlNodeType.XmlDeclaration:
							if (this.DeclarationGet != null)
							{
								this.DeclarationGet(this, new DeclarationEventArgs(xmlReader.Value));
							}
							break;
						}
						break;
					}
				}
				if (this.ParseEnd != null)
				{
					this.ParseEnd(this, new EventArgs());
				}
			}
		}

		// Token: 0x14000001 RID: 1
		// (add) Token: 0x0600004F RID: 79 RVA: 0x000025DC File Offset: 0x000007DC
		// (remove) Token: 0x06000050 RID: 80 RVA: 0x00002614 File Offset: 0x00000814
		public event EventHandler ElementStart;

		// Token: 0x14000002 RID: 2
		// (add) Token: 0x06000051 RID: 81 RVA: 0x0000264C File Offset: 0x0000084C
		// (remove) Token: 0x06000052 RID: 82 RVA: 0x00002684 File Offset: 0x00000884
		public event EventHandler ElementEnd;

		// Token: 0x14000003 RID: 3
		// (add) Token: 0x06000053 RID: 83 RVA: 0x000026BC File Offset: 0x000008BC
		// (remove) Token: 0x06000054 RID: 84 RVA: 0x000026F4 File Offset: 0x000008F4
		public event EventHandler AttributesGet;

		// Token: 0x14000004 RID: 4
		// (add) Token: 0x06000055 RID: 85 RVA: 0x0000272C File Offset: 0x0000092C
		// (remove) Token: 0x06000056 RID: 86 RVA: 0x00002764 File Offset: 0x00000964
		public event EventHandler ValueGet;

		// Token: 0x14000005 RID: 5
		// (add) Token: 0x06000057 RID: 87 RVA: 0x0000279C File Offset: 0x0000099C
		// (remove) Token: 0x06000058 RID: 88 RVA: 0x000027D4 File Offset: 0x000009D4
		public event EventHandler DeclarationGet;

		// Token: 0x14000006 RID: 6
		// (add) Token: 0x06000059 RID: 89 RVA: 0x0000280C File Offset: 0x00000A0C
		// (remove) Token: 0x0600005A RID: 90 RVA: 0x00002844 File Offset: 0x00000A44
		public event EventHandler CDataGet;

		// Token: 0x14000007 RID: 7
		// (add) Token: 0x0600005B RID: 91 RVA: 0x0000287C File Offset: 0x00000A7C
		// (remove) Token: 0x0600005C RID: 92 RVA: 0x000028B4 File Offset: 0x00000AB4
		public event EventHandler ParseStart;

		// Token: 0x14000008 RID: 8
		// (add) Token: 0x0600005D RID: 93 RVA: 0x000028EC File Offset: 0x00000AEC
		// (remove) Token: 0x0600005E RID: 94 RVA: 0x00002924 File Offset: 0x00000B24
		public event EventHandler ParseEnd;

		// Token: 0x0600005F RID: 95 RVA: 0x0000295C File Offset: 0x00000B5C
		public static XElement Serialize(TAPRoot model)
		{
			XElement xelement = new XElement("add_points");
			foreach (KeyValuePair<string, string> keyValuePair in model.GetAllAttribute())
			{
				xelement.Add(new XAttribute(keyValuePair.Key, keyValuePair.Value));
			}
			return xelement;
		}
	}
}
